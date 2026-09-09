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

func TestProfileRenameRollsBackPartialSecretCopy(t *testing.T) {
	profiles := storage.NewMemoryProfileStore()
	backend := storage.NewMemorySecretStore()
	secrets := &failingSecretStore{backend: backend, failProfile: "new", failName: secretSession}
	manager := ProfileManager{Profiles: profiles, Secrets: secrets}
	ctx := context.Background()
	if _, err := manager.Create(ctx, "old", "Old"); err != nil {
		t.Fatal(err)
	}
	if err := manager.ImportCookies(ctx, "old", Cookies{"c_user": "1", "xs": "x"}); err != nil {
		t.Fatal(err)
	}
	if err := manager.SaveSession(ctx, "old", Session{FBID: "1", DTSG: "d", Jazoest: "j", SessionID: "s"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Rename(ctx, "old", "new"); err == nil {
		t.Fatal("expected rename failure")
	}
	if profile, err := profiles.Get(ctx, "old"); err != nil || profile.Name != "old" {
		t.Fatalf("old profile was not preserved: %#v %v", profile, err)
	}
	if _, err := profiles.Get(ctx, "new"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("new profile leaked after rollback: %v", err)
	}
	if cookies, err := manager.LoadCookies(ctx, "old"); err != nil || cookies["c_user"] != "1" {
		t.Fatalf("old cookies were not preserved: %#v %v", cookies, err)
	}
	if _, err := backend.Get(ctx, "new", secretCookies); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("new cookies leaked after rollback: %v", err)
	}
}

type failingSecretStore struct {
	backend     storage.SecretStore
	failProfile string
	failName    string
}

func (s *failingSecretStore) Get(ctx context.Context, profile, name string) ([]byte, error) {
	return s.backend.Get(ctx, profile, name)
}

func (s *failingSecretStore) Put(ctx context.Context, profile, name string, value []byte) error {
	if profile == s.failProfile && name == s.failName {
		return errors.New("injected secret-store failure")
	}
	return s.backend.Put(ctx, profile, name, value)
}

func (s *failingSecretStore) Delete(ctx context.Context, profile, name string) error {
	return s.backend.Delete(ctx, profile, name)
}
