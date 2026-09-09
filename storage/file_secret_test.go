package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSecretStoreLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	store := NewFileSecretStore(path)
	ctx := context.Background()
	if err := store.Put(ctx, "default", "cookies", []byte("c_user=1; xs=x")); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "default", "cookies")
	if err != nil || string(got) != "c_user=1; xs=x" {
		t.Fatalf("unexpected secret: %q %v", got, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "c_user=1; xs=x" {
		t.Fatal("secret file unexpectedly stored bare plaintext")
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("secret store permissions are not private: %v %v", info, err)
	}
	if err := store.Delete(ctx, "default", "cookies"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "default", "cookies"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestFileSecretStoreRejectsInvalidKeys(t *testing.T) {
	store := NewFileSecretStore(filepath.Join(t.TempDir(), "secrets.json"))
	for _, test := range []struct{ profile, name string }{{"", "cookies"}, {"default", ""}, {"bad\x00profile", "cookies"}, {"default", "bad\x00name"}} {
		if err := store.Put(context.Background(), test.profile, test.name, []byte("x")); !errors.Is(err, ErrInvalidSecretKey) {
			t.Fatalf("expected invalid secret key for %q/%q, got %v", test.profile, test.name, err)
		}
	}
}
