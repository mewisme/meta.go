package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParseCookieString(t *testing.T) {
	got := ParseCookieString("c_user=123; xs=abc; malformed; fr=a=b")
	if got["c_user"] != "123" || got["xs"] != "abc" || got["fr"] != "a=b" {
		t.Fatalf("unexpected cookies: %#v", got)
	}
}

func TestDecodeJSONObject(t *testing.T) {
	var payload map[string]int
	if err := DecodeJSONObject([]byte("for (;;); {\"value\": 1}"), true, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["value"] != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if err := DecodeJSONObject([]byte("[]"), false, &payload); err == nil {
		t.Fatal("expected object-only validation")
	}
}

func TestFormRequestAndHeaders(t *testing.T) {
	req, err := NewFormRequest(context.Background(), http.MethodPost, "https://example.com", url.Values{"a": {"1"}})
	if err != nil {
		t.Fatal(err)
	}
	ApplyBrowserHeaders(req, "https://example.com", "https://example.com/home")
	if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || req.Header.Get("Origin") == "" {
		t.Fatalf("unexpected headers: %#v", req.Header)
	}
}

func TestSafeRedirectPolicy(t *testing.T) {
	previous, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	next, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err := SafeRedirectPolicy(next, []*http.Request{previous}); err == nil {
		t.Fatal("expected HTTPS downgrade rejection")
	}
}

func TestClientDefaultsAndBodyLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "12345")
	}))
	defer server.Close()
	client := NewClient(nil)
	if client.HTTP.Jar == nil || client.HTTP.Timeout == 0 || client.HTTP.CheckRedirect == nil {
		t.Fatal("expected safe HTTP defaults")
	}
	client.MaxBodyBytes = 4
	req, err := NewRequest(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Do(req); err == nil {
		t.Fatal("expected response size rejection")
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestRetryClassification(t *testing.T) {
	if !IsRetryable(nil, http.StatusTooManyRequests) || !IsRetryable(timeoutError{}, 0) {
		t.Fatal("expected retryable condition")
	}
	if IsRetryable(errors.New("permanent"), http.StatusBadRequest) {
		t.Fatal("unexpected retry classification")
	}
}

func FuzzParseCookieString(f *testing.F) {
	f.Add("c_user=123; xs=abc")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		cookies := ParseCookieString(input)
		for key := range cookies {
			if strings.TrimSpace(key) == "" {
				t.Fatal("empty cookie key")
			}
		}
	})
}

func FuzzDecodeJSONObject(f *testing.F) {
	f.Add([]byte("for (;;);{\"ok\":true}"), true)
	f.Add([]byte("{}"), false)
	f.Fuzz(func(t *testing.T, data []byte, strip bool) {
		var payload map[string]json.RawMessage
		_ = DecodeJSONObject(data, strip, &payload)
	})
}
