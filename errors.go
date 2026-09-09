package fbgo

import fberrors "go.mewis.me/fbgo/errors"

var (
	ErrUnauthorized       = fberrors.ErrUnauthorized
	ErrSessionExpired     = fberrors.ErrSessionExpired
	ErrCheckpointRequired = fberrors.ErrCheckpointRequired
	ErrRateLimited        = fberrors.ErrRateLimited
	ErrInvalidInput       = fberrors.ErrInvalidInput
	ErrNotConnected       = fberrors.ErrNotConnected
	ErrE2EENotReady       = fberrors.ErrE2EENotReady
	ErrUnsupported        = fberrors.ErrUnsupported
	ErrProtocolChanged    = fberrors.ErrProtocolChanged
	ErrPermissionDenied   = fberrors.ErrPermissionDenied
)

type ProtocolError = fberrors.ProtocolError
type ErrorCategory = fberrors.ErrorCategory

const (
	ErrorCategoryUnknown    = fberrors.ErrorCategoryUnknown
	ErrorCategoryAuth       = fberrors.ErrorCategoryAuth
	ErrorCategoryCheckpoint = fberrors.ErrorCategoryCheckpoint
	ErrorCategoryRateLimit  = fberrors.ErrorCategoryRateLimit
	ErrorCategoryInput      = fberrors.ErrorCategoryInput
	ErrorCategoryConnection = fberrors.ErrorCategoryConnection
	ErrorCategoryE2EE       = fberrors.ErrorCategoryE2EE
	ErrorCategoryProtocol   = fberrors.ErrorCategoryProtocol
	ErrorCategoryPermission = fberrors.ErrorCategoryPermission
	ErrorCategoryNetwork    = fberrors.ErrorCategoryNetwork
	ErrorCategoryCanceled   = fberrors.ErrorCategoryCanceled
)

func ClassifyError(err error) ErrorCategory { return fberrors.Classify(err) }
