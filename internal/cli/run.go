package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"

	"go.mewis.me/meta.go/auth"
	"go.mewis.me/meta.go/config"
	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/storage"
)

type errorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type errorOutputEnvelope struct {
	OK    bool      `json:"ok"`
	Error errorBody `json:"error"`
}

func Run(args []string, in io.Reader, out, errOut io.Writer) int {
	cmd := New()
	cmd.SetArgs(args)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	err := cmd.Execute()
	if err == nil {
		return ExitOK
	}
	code := ExitCode(err)
	if jsonRequested(args) {
		enc := json.NewEncoder(errOut)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(errorOutputEnvelope{OK: false, Error: errorBody{Code: code, Message: err.Error()}})
	} else {
		_, _ = io.WriteString(errOut, err.Error()+"\n")
	}
	return code
}

func jsonRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || arg == "--json=true" || arg == "--jqo" || strings.HasPrefix(arg, "--jqo=") {
			return true
		}
	}
	return false
}

func classifyExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var netErr net.Error
	switch {
	case errors.Is(err, fberrors.ErrInvalidInput), errors.Is(err, config.ErrAmbiguousConfig), errors.Is(err, config.ErrUnsupportedFormat), errors.Is(err, storage.ErrNotFound), errors.Is(err, os.ErrNotExist), errors.Is(err, os.ErrExist), strings.Contains(err.Error(), "unknown flag"), strings.Contains(err.Error(), "unknown command"), strings.Contains(err.Error(), "arg(s)"):
		return ExitUsage
	case errors.Is(err, fberrors.ErrUnauthorized), errors.Is(err, fberrors.ErrSessionExpired), errors.Is(err, fberrors.ErrCheckpointRequired), errors.Is(err, auth.ErrInvalidCookies), errors.Is(err, auth.ErrTwoFactorRequired):
		return ExitAuth
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr):
		return ExitNetwork
	case errors.Is(err, fberrors.ErrE2EENotReady):
		return ExitE2EE
	case errors.Is(err, fberrors.ErrPermissionDenied), errors.Is(err, fberrors.ErrRateLimited):
		return ExitRemote
	case errors.Is(err, fberrors.ErrProtocolChanged), errors.Is(err, fberrors.ErrUnsupported):
		return ExitProtocol
	default:
		return ExitInternal
	}
}
