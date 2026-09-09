package fbgo

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestNewDefaults(t *testing.T) {
	client := New()
	if client.httpClient == nil || client.logger == nil {
		t.Fatal("expected default dependencies")
	}
	if client.timeout != 30*time.Second || client.httpClient.Timeout != 30*time.Second {
		t.Fatalf("unexpected default timeout: %s / %s", client.timeout, client.httpClient.Timeout)
	}
}

func TestNewOptions(t *testing.T) {
	httpClient := &http.Client{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := New(WithHTTPClient(httpClient), WithLogger(logger), WithTimeout(5*time.Second))
	if client.httpClient != httpClient || client.logger == nil || client.timeout != 5*time.Second {
		t.Fatal("options were not applied")
	}
	if httpClient.Timeout != 5*time.Second {
		t.Fatalf("HTTP timeout was not propagated: %s", httpClient.Timeout)
	}
}
