package auth

import (
	"context"
	"errors"
	"testing"

	"go.mewis.me/fbgo/storage"
)

func TestProfileLifecycle(t *testing.T) {
	profiles := storage.NewMemoryProfileStore()
	secrets := storage.NewMemorySecretStore()
	manager := ProfileManager{Profiles: profiles, Secrets: secrets}
	ctx := context.Background()
	if _, err := manager.Create(ctx, "default", "Default"); err != nil {
		t.Fatal(err)
	}
	if err := manager.ImportCookies(ctx, "default", Cookies{"c_user": "123", "xs": "x"}); err != nil {
		t.Fatal(err)
	}
	loaded, err := manager.LoadCookies(ctx, "default")
	if err != nil || loaded["c_user"] != "123" {
		t.Fatalf("unexpected cookies: %#v %v", loaded, err)
	}
	if err := manager.Logout(ctx, "default", LogoutPolicy{RemoveCookies: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.LoadCookies(ctx, "default"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected removed cookie secret, got %v", err)
	}
}

func TestLegacyImportDoesNotOverwrite(t *testing.T) {
	profiles := storage.NewMemoryProfileStore()
	secrets := storage.NewMemorySecretStore()
	manager := ProfileManager{Profiles: profiles, Secrets: secrets}
	ctx := context.Background()
	_, _ = manager.Create(ctx, "default", "")
	if err := manager.ImportLegacyCookieString(ctx, "default", "c_user=1; xs=a"); err != nil {
		t.Fatal(err)
	}
	if err := manager.ImportLegacyCookieString(ctx, "default", "c_user=2; xs=b"); err == nil {
		t.Fatal("expected overwrite refusal")
	}
}

func TestLegacyJSONImport(t *testing.T) {
	profiles := storage.NewMemoryProfileStore()
	secrets := storage.NewMemorySecretStore()
	manager := ProfileManager{Profiles: profiles, Secrets: secrets}
	ctx := context.Background()
	_, _ = manager.Create(ctx, "legacy", "")
	if err := manager.ImportLegacyJSON(ctx, "legacy", []byte(`{"cookieFacebook":"c_user=1; xs=a"}`)); err != nil {
		t.Fatal(err)
	}
	cookies, err := manager.LoadCookies(ctx, "legacy")
	if err != nil || cookies["c_user"] != "1" {
		t.Fatalf("unexpected imported cookies: %#v %v", cookies, err)
	}
}

func TestLegacyE2EEImportDoesNotOverwrite(t *testing.T) {
	profiles := storage.NewMemoryProfileStore()
	secrets := storage.NewMemorySecretStore()
	manager := ProfileManager{Profiles: profiles, Secrets: secrets}
	ctx := context.Background()
	_, _ = manager.Create(ctx, "legacy", "")
	if err := manager.ImportLegacyE2EEState(ctx, "legacy", []byte(`{"device":"state"}`)); err != nil {
		t.Fatal(err)
	}
	if err := manager.ImportLegacyE2EEState(ctx, "legacy", []byte(`{"device":"new"}`)); err == nil {
		t.Fatal("expected E2EE state overwrite refusal")
	}
}

func TestProfileRenameMovesSecrets(t *testing.T) {
	profiles := storage.NewMemoryProfileStore()
	secrets := storage.NewMemorySecretStore()
	manager := ProfileManager{Profiles: profiles, Secrets: secrets}
	ctx := context.Background()
	_, _ = manager.Create(ctx, "old", "Old")
	if err := manager.ImportCookies(ctx, "old", Cookies{"c_user": "1", "xs": "x"}); err != nil {
		t.Fatal(err)
	}
	renamed, err := manager.Rename(ctx, "old", "new")
	if err != nil || renamed.Name != "new" {
		t.Fatalf("rename failed: %#v %v", renamed, err)
	}
	if _, err := manager.LoadCookies(ctx, "old"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("old secret still exists: %v", err)
	}
	if cookies, err := manager.LoadCookies(ctx, "new"); err != nil || cookies["c_user"] != "1" {
		t.Fatalf("new secret missing: %#v %v", cookies, err)
	}
}
