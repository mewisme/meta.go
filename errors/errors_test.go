package fberrors

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		err  error
		want ErrorCategory
	}{
		{ErrUnauthorized, ErrorCategoryAuth},
		{ErrSessionExpired, ErrorCategoryAuth},
		{ErrCheckpointRequired, ErrorCategoryCheckpoint},
		{ErrRateLimited, ErrorCategoryRateLimit},
		{ErrInvalidInput, ErrorCategoryInput},
		{ErrAccountMismatch, ErrorCategoryInput},
		{ErrNotConnected, ErrorCategoryConnection},
		{ErrE2EENotReady, ErrorCategoryE2EE},
		{ErrProtocolChanged, ErrorCategoryProtocol},
		{ErrPermissionDenied, ErrorCategoryPermission},
		{context.Canceled, ErrorCategoryCanceled},
		{context.DeadlineExceeded, ErrorCategoryNetwork},
		{&net.DNSError{Err: "timeout", Name: "example.com", IsTimeout: true}, ErrorCategoryNetwork},
		{errors.New("other"), ErrorCategoryUnknown},
	}
	for _, test := range tests {
		if got := Classify(test.err); got != test.want {
			t.Fatalf("Classify(%v) = %s, want %s", test.err, got, test.want)
		}
	}
}

func TestProtocolErrorClassificationUsesCause(t *testing.T) {
	err := &ProtocolError{Operation: "query", Cause: ErrProtocolChanged}
	if got := Classify(err); got != ErrorCategoryProtocol {
		t.Fatalf("category = %s", got)
	}
}
