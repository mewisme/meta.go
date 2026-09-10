package fberrors

import (
	"context"
	"errors"
	"fmt"
	"net"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionExpired     = errors.New("session expired")
	ErrCheckpointRequired = errors.New("checkpoint required")
	ErrRateLimited        = errors.New("rate limited")
	ErrInvalidInput       = errors.New("invalid input")
	ErrAccountMismatch    = errors.New("account mismatch")
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

type ErrorCategory string

const (
	ErrorCategoryUnknown    ErrorCategory = "unknown"
	ErrorCategoryAuth       ErrorCategory = "auth"
	ErrorCategoryCheckpoint ErrorCategory = "checkpoint"
	ErrorCategoryRateLimit  ErrorCategory = "rate_limit"
	ErrorCategoryInput      ErrorCategory = "invalid_input"
	ErrorCategoryConnection ErrorCategory = "connection"
	ErrorCategoryE2EE       ErrorCategory = "e2ee"
	ErrorCategoryProtocol   ErrorCategory = "protocol"
	ErrorCategoryPermission ErrorCategory = "permission"
	ErrorCategoryNetwork    ErrorCategory = "network"
	ErrorCategoryCanceled   ErrorCategory = "canceled"
)

// Classify returns a stable, low-cardinality category suitable for health and metrics labels.
func Classify(err error) ErrorCategory {
	if err == nil {
		return ErrorCategoryUnknown
	}
	var netErr net.Error
	switch {
	case errors.Is(err, context.Canceled):
		return ErrorCategoryCanceled
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr):
		return ErrorCategoryNetwork
	case errors.Is(err, ErrCheckpointRequired):
		return ErrorCategoryCheckpoint
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrSessionExpired):
		return ErrorCategoryAuth
	case errors.Is(err, ErrRateLimited):
		return ErrorCategoryRateLimit
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrAccountMismatch):
		return ErrorCategoryInput
	case errors.Is(err, ErrNotConnected):
		return ErrorCategoryConnection
	case errors.Is(err, ErrE2EENotReady):
		return ErrorCategoryE2EE
	case errors.Is(err, ErrProtocolChanged), errors.Is(err, ErrUnsupported):
		return ErrorCategoryProtocol
	case errors.Is(err, ErrPermissionDenied):
		return ErrorCategoryPermission
	default:
		return ErrorCategoryUnknown
	}
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
