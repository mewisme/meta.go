package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"go.mau.fi/mautrix-meta/pkg/messagix"
	fberrors "go.mewis.me/fbgo/errors"
	"maunium.net/go/mautrix/bridgev2"
)

func TestNormalizeCredentialLoginError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"checkpoint", messagix.ErrCheckpointRequired, fberrors.ErrCheckpointRequired},
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
