package cli

import (
	"errors"
	"testing"

	fberrors "go.mewis.me/fbgo/errors"
)

func TestLoginInputJSONContract(t *testing.T) {
	var input loginInput
	if err := decodeJSONInput([]byte(`{"identifier":"user@example.com","password":"secret","totp":"JBSWY3DPEHPK3PXP"}`), "", &input); err != nil {
		t.Fatal(err)
	}
	if input.Identifier != "user@example.com" || input.Password != "secret" || input.TOTP != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("unexpected input: %#v", input)
	}
	for _, data := range [][]byte{
		[]byte(`{"email":"user@example.com"}`),
		[]byte(`{"identifier":"user@example.com","unknown":true}`),
	} {
		if err := decodeJSONInput(data, "", &input); err == nil {
			t.Fatalf("expected strict decode failure for %s", data)
		}
	}
}

func TestLoginInputJQSelector(t *testing.T) {
	data := []byte(`{"account":{"identifier":"user@example.com","password":"secret","otp":"123456"}}`)
	var input loginInput
	if err := decodeJSONInput(data, ".account", &input); err != nil {
		t.Fatal(err)
	}
	if input.Identifier != "user@example.com" || input.Password != "secret" || input.OTP != "123456" {
		t.Fatalf("unexpected jq-selected input: %#v", input)
	}
}

func TestValidateLoginInput(t *testing.T) {
	for _, input := range []loginInput{{OTP: "123456"}, {TOTP: "JBSWY3DPEHPK3PXP"}} {
		if err := validateLoginInput(input); err != nil {
			t.Fatal(err)
		}
	}
	for _, input := range []loginInput{{OTP: "12345"}, {OTP: "abcdef"}, {TOTP: "bad!"}, {TOTP: "JBSWY3DPEHPK3PXP", OTP: "123456"}} {
		if err := validateLoginInput(input); !errors.Is(err, fberrors.ErrInvalidInput) {
			t.Fatalf("expected invalid input for %#v, got %v", input, err)
		}
	}
}

func TestCLILoginCredentialFlags(t *testing.T) {
	root := New()
	cmd, _, err := root.Find([]string{"auth", "login"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"identifier", "password", "totp", "otp", "input"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing auth login flag --%s", name)
		}
	}
	for _, name := range []string{"jqi", "jqo"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Fatalf("missing global --%s flag", name)
		}
	}
	if root.PersistentFlags().Lookup("jq") != nil {
		t.Fatal("ambiguous global --jq flag must not be exposed")
	}
}
