package auth

import (
	"errors"
	"testing"
)

func TestParseCookieStringAndValidation(t *testing.T) {
	cookies, err := ParseCookieString("xs=abc; c_user=123; fr=a=b; malformed")
	if err != nil {
		t.Fatal(err)
	}
	if cookies["fr"] != "a=b" || cookies["c_user"] != "123" {
		t.Fatalf("unexpected cookies: %#v", cookies)
	}
	if err := cookies.ValidateRegular(); err != nil {
		t.Fatal(err)
	}
}

func TestCookieValidationRejectsMissingRequired(t *testing.T) {
	if err := (Cookies{"c_user": "123"}).ValidateRegular(); !errors.Is(err, ErrInvalidCookies) {
		t.Fatalf("expected invalid cookies, got %v", err)
	}
}

func TestParseBrowserCookieJSON(t *testing.T) {
	cookies, err := ParseBrowserCookieJSON([]byte(`[{"name":"c_user","value":"123"},{"name":"xs","value":"abc"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := cookies.ValidateE2EE(); err != nil {
		t.Fatal(err)
	}
}
