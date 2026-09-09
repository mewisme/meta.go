package auth

import (
	"testing"
	"time"
)

func TestTOTPReferenceVector(t *testing.T) {
	code, err := TOTP("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Unix(59, 0))
	if err != nil {
		t.Fatal(err)
	}
	if code != "287082" {
		t.Fatalf("unexpected code: %s", code)
	}
}

func TestResolveDirectOTP(t *testing.T) {
	code, err := ResolveOTP("123456", time.Now())
	if err != nil || code != "123456" {
		t.Fatalf("unexpected direct OTP: %q %v", code, err)
	}
}
