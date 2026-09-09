package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactingHandler(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewRedactingHandler(slog.NewTextHandler(&output, nil)))
	logger.Info("request", "token", "SECRET_TOKEN", "safe", "visible", "auth", slog.GroupValue(slog.String("cookie", "SECRET_COOKIE")))
	got := output.String()
	if strings.Contains(got, "SECRET_TOKEN") || strings.Contains(got, "SECRET_COOKIE") {
		t.Fatalf("secret leaked in log: %s", got)
	}
	if !strings.Contains(got, Redacted) || !strings.Contains(got, "visible") {
		t.Fatalf("unexpected redacted log: %s", got)
	}
}

func TestSensitiveKeys(t *testing.T) {
	for _, key := range []string{"token", "access_token", "Cookie", "fb_dtsg", "session_id", "totp_seed"} {
		if !IsSensitiveKey(key) {
			t.Fatalf("expected %q to be sensitive", key)
		}
	}
	if IsSensitiveKey("thread_id") {
		t.Fatal("thread_id must not be treated as a secret")
	}
}
