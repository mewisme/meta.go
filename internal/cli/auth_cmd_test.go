package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"go.mewis.me/fbgo/auth"
)

func TestDecodeLoginInputStrict(t *testing.T) {
	input, err := decodeLoginInput([]byte(`{"email":"user@example.com","password":"secret","totp":"123456"}`))
	if err != nil {
		t.Fatal(err)
	}
	if input.Email != "user@example.com" || input.Password != "secret" || input.TOTP != "123456" {
		t.Fatalf("unexpected input: %#v", input)
	}
	for _, data := range [][]byte{
		[]byte(`{"credentials":{"email":"user@example.com"}}`),
		[]byte(`{"email":"user@example.com","unknown":true}`),
		[]byte(`{"email":"user@example.com"} {"password":"secret"}`),
	} {
		if _, err := decodeLoginInput(data); err == nil {
			t.Fatalf("expected strict decode failure for %s", data)
		}
	}
}

func TestResolveLoginFieldUsesPresetCredentials(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("unused\n"))
	cmd.SetErr(&bytes.Buffer{})
	preset := loginInput{Email: "user@example.com", Password: "password-value", TOTP: "123456"}
	tests := []struct {
		field auth.LoginField
		want  string
	}{
		{auth.LoginField{ID: "email"}, preset.Email},
		{auth.LoginField{ID: "password", Secret: true}, preset.Password},
		{auth.LoginField{ID: "two_factor_code", Secret: true}, preset.TOTP},
	}
	for _, test := range tests {
		got, err := resolveLoginField(cmd, test.field, preset)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("field %s: got %q want %q", test.field.ID, got, test.want)
		}
	}
}

func TestResolveLoginFieldPrefersAuthenticatorForTOTP(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("unused\n"))
	cmd.SetErr(&bytes.Buffer{})
	field := auth.LoginField{ID: "mfatype", Options: []string{"Text message", "Authentication app", "Email"}}
	got, err := resolveLoginField(cmd, field, loginInput{TOTP: "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Authentication app" {
		t.Fatalf("unexpected MFA method: %q", got)
	}
}

func TestCLIExposesNoSecretValueFlags(t *testing.T) {
	root := New()
	for _, path := range [][]string{{"auth", "login"}, {"auth", "import"}, {"cookies", "import"}} {
		cmd, _, err := root.Find(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"password", "cookie", "cookies", "totp", "otp", "secret"} {
			if cmd.Flags().Lookup(name) != nil || cmd.PersistentFlags().Lookup(name) != nil {
				t.Fatalf("%v exposes secret flag --%s", path, name)
			}
		}
	}
}
