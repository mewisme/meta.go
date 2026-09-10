package auth

import (
	"errors"
	"testing"

	fberrors "go.mewis.me/meta.go/errors"
)

func TestSourceValidate(t *testing.T) {
	credentials := Credentials{Identifier: "user", Password: "pw", OTP: "123456"}
	valid := []Source{
		{Cookies: Cookies{"c_user": "1", "xs": "x"}},
		{AppState: AppState{{Key: "c_user", Value: "1"}, {Key: "xs", Value: "x"}}},
		{Credentials: &credentials},
	}
	for _, source := range valid {
		if err := source.Validate(); err != nil {
			t.Fatalf("expected valid source %#v: %v", source, err)
		}
	}
	for _, source := range []Source{{}, {Cookies: Cookies{"c_user": "1", "xs": "x"}, Credentials: &credentials}} {
		if err := source.Validate(); !errors.Is(err, fberrors.ErrInvalidInput) {
			t.Fatalf("expected invalid source %#v: %v", source, err)
		}
	}
}
