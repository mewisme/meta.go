package fberrors

import (
	"errors"
	"fmt"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionExpired     = errors.New("session expired")
	ErrCheckpointRequired = errors.New("checkpoint required")
	ErrRateLimited        = errors.New("rate limited")
	ErrInvalidInput       = errors.New("invalid input")
	ErrNotConnected       = errors.New("not connected")
	ErrE2EENotReady       = errors.New("e2ee not ready")
	ErrUnsupported        = errors.New("unsupported")
	ErrProtocolChanged    = errors.New("protocol changed")
	ErrPermissionDenied   = errors.New("permission denied")
)

type ProtocolError struct {
	Operation  string
	Endpoint   string
	Code       string
	Subcode    string
	FBTraceID  string
	StatusCode int
	Retryable  bool
	Cause      error
}

func (e *ProtocolError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.StatusCode != 0 {
		return fmt.Sprintf("facebook protocol error during %s: HTTP %d", e.Operation, e.StatusCode)
	}
	return fmt.Sprintf("facebook protocol error during %s", e.Operation)
}

func (e *ProtocolError) Unwrap() error { return e.Cause }
