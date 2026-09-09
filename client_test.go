package fbgo

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mewis.me/fbgo/auth"
	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/storage"
)

func TestNewClientValidatesOptions(t *testing.T) {
	for _, option := range []Option{WithTimeout(0), WithTimeout(-time.Second), WithEventBuffer(0), WithEventBuffer(-1)} {
		if _, err := NewClient(option); !errors.Is(err, fberrors.ErrInvalidInput) {
			t.Fatalf("expected invalid input, got %v", err)
		}
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
