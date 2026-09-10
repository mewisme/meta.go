package server

import (
	"errors"
	"testing"
	"time"

	"go.mewis.me/meta.go/auth"
	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
)

func TestAuthSourceFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input *metav1.SessionAuth
		check func(auth.Source) bool
	}{
		{"cookies", &metav1.SessionAuth{Source: &metav1.SessionAuth_Cookies{Cookies: &metav1.CookieMap{Values: map[string]string{"c_user": "1", "xs": "x"}}}}, func(source auth.Source) bool { return source.Cookies["xs"] == "x" }},
		{"appstate", &metav1.SessionAuth{Source: &metav1.SessionAuth_AppState{AppState: &metav1.AppState{Cookies: []*metav1.AppStateCookie{{Key: "c_user", Value: "1", Domain: ".facebook.com"}, {Key: "xs", Value: "x", HttpOnly: true}}}}}, func(source auth.Source) bool {
			return len(source.AppState) == 2 && source.AppState[0].Domain == ".facebook.com" && source.AppState[1].HTTPOnly
		}},
		{"otp", &metav1.SessionAuth{Source: &metav1.SessionAuth_Credentials{Credentials: &metav1.Credentials{Identifier: "user", Password: "pw", SecondFactor: &metav1.Credentials_Otp{Otp: "123456"}}}}, func(source auth.Source) bool { return source.Credentials != nil && source.Credentials.OTP == "123456" }},
		{"totp", &metav1.SessionAuth{Source: &metav1.SessionAuth_Credentials{Credentials: &metav1.Credentials{Identifier: "user", Password: "pw", SecondFactor: &metav1.Credentials_Totp{Totp: "JBSWY3DPEHPK3PXP"}}}}, func(source auth.Source) bool {
			return source.Credentials != nil && source.Credentials.TOTP == "JBSWY3DPEHPK3PXP"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, err := authSourceFromProto(test.input)
			if err != nil || !test.check(source) {
				t.Fatalf("unexpected conversion: %#v %v", source, err)
			}
		})
	}
}

func TestAuthSourceFromProtoRejectsInvalid(t *testing.T) {
	for _, input := range []*metav1.SessionAuth{
		nil,
		{},
		{Source: &metav1.SessionAuth_Cookies{Cookies: &metav1.CookieMap{Values: map[string]string{"c_user": "1"}}}},
		{Source: &metav1.SessionAuth_AppState{AppState: &metav1.AppState{}}},
		{Source: &metav1.SessionAuth_Credentials{Credentials: &metav1.Credentials{Identifier: "user", Password: "pw"}}},
	} {
		if _, err := authSourceFromProto(input); !errors.Is(err, fberrors.ErrInvalidInput) {
			t.Fatalf("expected invalid input for %#v, got %v", input, err)
		}
	}
}

func TestAuthSnapshotToProto(t *testing.T) {
	refreshedAt := time.Unix(1_700_000_000, 0).UTC()
	snapshot := auth.AuthSnapshot{
		Cookies:  auth.Cookies{"c_user": "1", "xs": "x"},
		AppState: auth.AppState{{Key: "c_user", Value: "1", Domain: ".facebook.com"}, {Key: "xs", Value: "x", HTTPOnly: true}},
		Session:  auth.Session{FBID: "1", Name: "Mew", Username: "mew", DTSG: "d", Jazoest: "j", LSD: "l", SessionID: "s", ClientRevision: 42, BootstrappedAt: refreshedAt},
	}
	got := authSnapshotToProto(snapshot)
	if got.GetCookies().GetValues()["xs"] != "x" || len(got.GetAppState().GetCookies()) != 2 || got.GetSession().GetAccountId() != "1" || got.GetSession().GetLsd() != "l" || got.GetSession().GetClientRevision() != 42 || !got.GetSession().GetRefreshedAt().AsTime().Equal(refreshedAt) {
		t.Fatalf("unexpected snapshot: %#v", got)
	}
}
