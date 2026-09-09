package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	fberrors "go.mewis.me/fbgo/errors"
	metaHTTP "go.mewis.me/meta-extra/pkg/messagix/httpclient"
	"maunium.net/go/mautrix/bridgev2"
)

func TestNormalizeCredentialLoginError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"checkpoint", metaHTTP.ErrCheckpointRequired, fberrors.ErrCheckpointRequired},
		{"bad credentials", errors.New("Invalid username or password"), fberrors.ErrUnauthorized},
		{"response rejection", bridgev2.RespError{ErrCode: "FI.MAU.META_LOGIN", Err: "rejected", StatusCode: 400}, fberrors.ErrUnauthorized},
		{"phone input", bridgev2.RespError{ErrCode: "FI.MAU.META_PHONE_NUMBER", Err: "phone unsupported", StatusCode: 400}, fberrors.ErrInvalidInput},
		{"protocol", errors.New("unexpected bloks state"), fberrors.ErrProtocolChanged},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := normalizeCredentialLoginError(test.err); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
	if err := normalizeCredentialLoginError(context.DeadlineExceeded); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline changed: %v", err)
	}
	netErr := &net.DNSError{Err: "timeout", Name: "facebook.com", IsTimeout: true}
	if err := normalizeCredentialLoginError(netErr); !errors.Is(err, netErr) {
		t.Fatalf("network error changed: %v", err)
	}
	if err := normalizeCredentialLoginError(nil); err != nil {
		t.Fatalf("nil became %v", err)
	}
}

func TestLoginResponseErrorWrapped(t *testing.T) {
	original := bridgev2.RespError{ErrCode: "FI.MAU.META_LOGIN", Err: "rejected"}
	got := loginResponseError(fmt.Errorf("wrapped: %w", original))
	if got == nil || got.ErrCode != original.ErrCode {
		t.Fatalf("unexpected response error: %#v", got)
	}
}

func TestFB4APasswordFormContract(t *testing.T) {
	state := &fb4aState{credentials: Credentials{Identifier: "user"}, apiKey: "key", appToken: "token", deviceID: "device", adID: "device", secureID: "device", machineID: "machine"}
	form := state.form("pw", "password", 1)
	for key, want := range map[string]string{"email": "user", "password": "pw", "credentials_type": "password", "try_num": "1", "jazoest": "22421", "api_key": "key", "access_token": "token"} {
		if form.Get(key) != want {
			t.Fatalf("%s = %q, want %q", key, form.Get(key), want)
		}
	}
}

func TestFB4ATwoFactorFormContract(t *testing.T) {
	state := &fb4aState{credentials: Credentials{Identifier: "user"}, deviceID: "device", adID: "device", secureID: "device", machineID: "machine"}
	form := state.twoFactorForm("123456", "123456", "10001", "factor", 2)
	for key, want := range map[string]string{"credentials_type": "two_factor", "password": "123456", "twofactor_code": "123456", "userid": "10001", "first_factor": "factor", "jazoest": "22327", "try_num": "2"} {
		if form.Get(key) != want {
			t.Fatalf("%s = %q, want %q", key, form.Get(key), want)
		}
	}
}

func TestFB4ATwoFactorMetadata(t *testing.T) {
	for _, raw := range []json.RawMessage{json.RawMessage(`{"uid":"10001","login_first_factor":"factor"}`), json.RawMessage(`"{\"userid\":\"10001\",\"first_factor\":\"factor\"}"`)} {
		uid, factor := fb4aTwoFactorMetadata(raw)
		if uid != "10001" || factor != "factor" {
			t.Fatalf("unexpected metadata: %q %q", uid, factor)
		}
	}
}

func TestCredentialLoginFallbackPolicy(t *testing.T) {
	credentials := Credentials{Identifier: "user", Password: "pw"}
	fallbackCalls := 0
	fallback := func(context.Context, Credentials) (Cookies, error) {
		fallbackCalls++
		return Cookies{"c_user": "1", "xs": "x"}, nil
	}
	protocolStep := func(context.Context, map[string]string) (*loginChallenge, Cookies, error) {
		return nil, nil, &fberrors.ProtocolError{Operation: "test", Cause: fberrors.ErrProtocolChanged}
	}
	if _, err := loginCredentials(context.Background(), credentials, protocolStep, fallback); err != nil {
		t.Fatal(err)
	}
	if fallbackCalls != 1 {
		t.Fatalf("fallback calls = %d", fallbackCalls)
	}
	unauthorizedStep := func(context.Context, map[string]string) (*loginChallenge, Cookies, error) {
		return nil, nil, fberrors.ErrUnauthorized
	}
	if _, err := loginCredentials(context.Background(), credentials, unauthorizedStep, fallback); !errors.Is(err, fberrors.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	if fallbackCalls != 1 {
		t.Fatalf("fallback called after auth rejection: %d", fallbackCalls)
	}
}

func TestCredentialValidation(t *testing.T) {
	for _, credentials := range []Credentials{{}, {Identifier: "user"}, {Identifier: "user", Password: "pw", TOTP: "bad!"}, {Identifier: "user", Password: "pw", OTP: "12345"}, {Identifier: "user", Password: "pw", TOTP: "JBSWY3DPEHPK3PXP", OTP: "123456"}} {
		if err := validateCredentials(credentials); !errors.Is(err, fberrors.ErrInvalidInput) {
			t.Fatalf("expected invalid input for %#v, got %v", credentials, err)
		}
	}
}

func TestFB4AURLValuesRemainFormEncoded(t *testing.T) {
	form := url.Values{"identifier": {"a+b@example.com"}}
	parsed, err := url.ParseQuery(form.Encode())
	if err != nil || parsed.Get("identifier") != "a+b@example.com" {
		t.Fatalf("form encoding mismatch: %#v %v", parsed, err)
	}
}

func TestFB4ALoginHTTPFlow(t *testing.T) {
	forms := make([]url.Values, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		data, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(data))
		forms = append(forms, form)
		w.Header().Set("Content-Type", "application/json")
		if len(forms) == 1 {
			_, _ = io.WriteString(w, `{"error":{"error_subcode":1348162,"error_data":{"uid":"10001","login_first_factor":"factor"}}}`)
			return
		}
		_, _ = io.WriteString(w, `{"session_cookies":[{"name":"c_user","value":"10001"},{"name":"xs","value":"x"}]}`)
	}))
	defer server.Close()
	cookies, err := loginFB4A(context.Background(), Credentials{Identifier: "user", Password: "pw", OTP: "123456"}, fb4aConfig{HTTP: server.Client(), URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if cookies["c_user"] != "10001" || cookies["xs"] != "x" || len(forms) != 2 {
		t.Fatalf("unexpected result: cookies=%#v forms=%d", cookies, len(forms))
	}
	if forms[0].Get("try_num") != "1" || forms[1].Get("try_num") != "2" || forms[1].Get("twofactor_code") != "123456" {
		t.Fatalf("unexpected login sequence: %#v", forms)
	}
}
