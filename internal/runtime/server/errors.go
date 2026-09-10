package server

import (
	"context"
	"errors"
	"net"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
)

func grpcError(err error) error {
	if err == nil {
		return nil
	}
	category := fberrors.Classify(err)
	code := grpcCode(err, category)
	detail := errorDetail(err)
	st := status.New(code, err.Error())
	withDetails, detailErr := st.WithDetails(detail)
	if detailErr != nil {
		return st.Err()
	}
	return withDetails.Err()
}

func errorDetail(err error) *metav1.ErrorDetail {
	if err == nil {
		return nil
	}
	category := fberrors.Classify(err)
	return &metav1.ErrorDetail{Category: protoErrorCategory(category), Code: stableErrorCode(err), Message: err.Error(), Retryable: retryable(err, category)}
}

func grpcCode(err error, category fberrors.ErrorCategory) codes.Code {
	switch {
	case errors.Is(err, context.Canceled):
		return codes.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return codes.DeadlineExceeded
	case errors.Is(err, session.ErrNotFound):
		return codes.NotFound
	case errors.Is(err, session.ErrClosed):
		return codes.FailedPrecondition
	case errors.Is(err, fberrors.ErrUnsupported):
		return codes.Unimplemented
	}
	switch category {
	case fberrors.ErrorCategoryAuth:
		return codes.Unauthenticated
	case fberrors.ErrorCategoryCheckpoint:
		return codes.FailedPrecondition
	case fberrors.ErrorCategoryRateLimit:
		return codes.ResourceExhausted
	case fberrors.ErrorCategoryInput:
		return codes.InvalidArgument
	case fberrors.ErrorCategoryConnection, fberrors.ErrorCategoryE2EE:
		return codes.FailedPrecondition
	case fberrors.ErrorCategoryProtocol:
		return codes.FailedPrecondition
	case fberrors.ErrorCategoryPermission:
		return codes.PermissionDenied
	case fberrors.ErrorCategoryNetwork:
		return codes.Unavailable
	case fberrors.ErrorCategoryCanceled:
		return codes.Canceled
	default:
		return codes.Unknown
	}
}

func stableErrorCode(err error) string {
	for _, item := range []struct {
		target error
		code   string
	}{
		{session.ErrNotFound, "session_not_found"}, {session.ErrClosed, "session_closed"},
		{fberrors.ErrUnauthorized, "unauthorized"}, {fberrors.ErrSessionExpired, "session_expired"},
		{fberrors.ErrCheckpointRequired, "checkpoint_required"}, {fberrors.ErrRateLimited, "rate_limited"},
		{fberrors.ErrInvalidInput, "invalid_input"}, {fberrors.ErrNotConnected, "not_connected"},
		{fberrors.ErrE2EENotReady, "e2ee_not_ready"}, {fberrors.ErrUnsupported, "unsupported"},
		{fberrors.ErrProtocolChanged, "protocol_changed"}, {fberrors.ErrPermissionDenied, "permission_denied"},
		{context.Canceled, "canceled"}, {context.DeadlineExceeded, "deadline_exceeded"},
	} {
		if errors.Is(err, item.target) {
			return item.code
		}
	}
	return "unknown"
}

func retryable(err error, category fberrors.ErrorCategory) bool {
	var protocolErr *fberrors.ProtocolError
	if errors.As(err, &protocolErr) && protocolErr.Retryable {
		return true
	}
	var netErr net.Error
	return category == fberrors.ErrorCategoryRateLimit || category == fberrors.ErrorCategoryNetwork || errors.As(err, &netErr)
}

func protoErrorCategory(category fberrors.ErrorCategory) metav1.ErrorCategory {
	switch category {
	case fberrors.ErrorCategoryAuth:
		return metav1.ErrorCategory_ERROR_CATEGORY_AUTH
	case fberrors.ErrorCategoryCheckpoint:
		return metav1.ErrorCategory_ERROR_CATEGORY_CHECKPOINT
	case fberrors.ErrorCategoryRateLimit:
		return metav1.ErrorCategory_ERROR_CATEGORY_RATE_LIMIT
	case fberrors.ErrorCategoryInput:
		return metav1.ErrorCategory_ERROR_CATEGORY_INVALID_INPUT
	case fberrors.ErrorCategoryConnection:
		return metav1.ErrorCategory_ERROR_CATEGORY_CONNECTION
	case fberrors.ErrorCategoryE2EE:
		return metav1.ErrorCategory_ERROR_CATEGORY_E2EE
	case fberrors.ErrorCategoryProtocol:
		return metav1.ErrorCategory_ERROR_CATEGORY_PROTOCOL
	case fberrors.ErrorCategoryPermission:
		return metav1.ErrorCategory_ERROR_CATEGORY_PERMISSION
	case fberrors.ErrorCategoryNetwork:
		return metav1.ErrorCategory_ERROR_CATEGORY_NETWORK
	case fberrors.ErrorCategoryCanceled:
		return metav1.ErrorCategory_ERROR_CATEGORY_CANCELED
	default:
		return metav1.ErrorCategory_ERROR_CATEGORY_UNKNOWN
	}
}
