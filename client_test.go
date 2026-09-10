package meta

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mewis.me/meta.go/auth"
	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/storage"
)

func TestNewClientValidatesOptions(t *testing.T) {
	for _, option := range []Option{WithTimeout(0), WithTimeout(-time.Second), WithEventBuffer(0), WithEventBuffer(-1)} {
		if _, err := NewClient(option); !errors.Is(err, fberrors.ErrInvalidInput) {
			t.Fatalf("expected invalid input, got %v", err)
		}
	}
}

func TestWithHTTPClientDoesNotMutateCallerClient(t *testing.T) {
	httpClient := &http.Client{Timeout: time.Minute}
	client, err := NewClient(WithHTTPClient(httpClient), WithTimeout(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if httpClient.Timeout != time.Minute {
		t.Fatalf("caller HTTP client was mutated: %s", httpClient.Timeout)
	}
	if client.httpClient == httpClient || client.httpClient.Timeout != 3*time.Second {
		t.Fatalf("client HTTP copy was not configured independently: %#v", client.httpClient)
	}
}

func TestWithLoggerReceivesLifecycleDiagnostics(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client, err := NewClient(WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "client closed") {
		t.Fatalf("custom logger did not receive lifecycle diagnostics: %q", output.String())
	}
}

func TestClientOnFiltersAndUnsubscribes(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	called := 0
	unsubscribe := client.On(EventMessage, func(Event) { called++ })
	client.dispatchEvent(Event{Kind: EventTyping})
	client.dispatchEvent(Event{Kind: EventMessage})
	if called != 1 {
		t.Fatalf("unexpected handler calls: %d", called)
	}
	unsubscribe()
	unsubscribe()
	client.dispatchEvent(Event{Kind: EventMessage})
	if called != 1 {
		t.Fatalf("handler called after unsubscribe: %d", called)
	}
}

func TestClientConnectRequiresCredentials(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Connect(context.Background()); !errors.Is(err, fberrors.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestClientLoadsCookiesFromSecretStore(t *testing.T) {
	secrets := storage.NewMemorySecretStore()
	manager := auth.ProfileManager{Profiles: storage.NewMemoryProfileStore(), Secrets: secrets}
	ctx := context.Background()
	if _, err := manager.Create(ctx, "default", ""); err != nil {
		t.Fatal(err)
	}
	if err := manager.ImportCookies(ctx, "default", auth.Cookies{"c_user": "1", "xs": "x"}); err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(WithProfile(storage.Profile{Name: "default"}), WithSecretStore(secrets))
	if err != nil {
		t.Fatal(err)
	}
	client.mu.RLock()
	profile := client.profile
	store := client.secrets
	client.mu.RUnlock()
	loaded, err := (auth.ProfileManager{Secrets: store}).LoadCookies(ctx, profile.Name)
	if err != nil || loaded["c_user"] != "1" {
		t.Fatalf("unexpected cookies: %#v %v", loaded, err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClientCloseCancelsBeforeWaitingForConnect(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	client.connectMu.Lock()
	done := make(chan struct{})
	go func() {
		_ = client.Close()
		close(done)
	}()
	select {
	case <-client.ctx.Done():
	case <-time.After(time.Second):
		client.connectMu.Unlock()
		t.Fatal("Close did not cancel the client before waiting for connect serialization")
	}
	select {
	case <-done:
		client.connectMu.Unlock()
		t.Fatal("Close returned while a serialized connect was still in flight")
	default:
	}
	client.connectMu.Unlock()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not finish after connect serialization was released")
	}
}

func TestClientCloseReleasesSensitiveReferences(t *testing.T) {
	secrets := storage.NewMemorySecretStore()
	client, err := NewClient(WithCookies(auth.Cookies{"c_user": "1", "xs": "secret"}), WithProfile(storage.Profile{Name: "default"}), WithSecretStore(secrets))
	if err != nil {
		t.Fatal(err)
	}
	client.On(EventMessage, func(Event) {})
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	client.mu.RLock()
	cookies, profile, store, account := client.cookies, client.profile, client.secrets, client.account
	client.mu.RUnlock()
	client.handlerMu.RLock()
	handlers := len(client.handlers)
	client.handlerMu.RUnlock()
	if cookies != nil || profile.Name != "" || store != nil || !account.ID.Empty() || handlers != 0 {
		t.Fatalf("sensitive references retained after close: cookies=%d profile=%q store=%t account=%s handlers=%d", len(cookies), profile.Name, store != nil, account.ID, handlers)
	}
}
