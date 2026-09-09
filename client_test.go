package fbgo

import (
	"context"
	"errors"
	"testing"

	"go.mewis.me/fbgo/auth"
	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/storage"
)

func TestClientConnectRequiresCredentials(t *testing.T) {
	client := New()
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
	client := New(WithProfile(storage.Profile{Name: "default"}), WithSecretStore(secrets))
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
