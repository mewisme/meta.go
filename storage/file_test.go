package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileProfileStoreLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	store := NewFileProfileStore(path)
	ctx := context.Background()
	profile := Profile{Name: "default", DisplayName: "Default", Metadata: map[string]string{"region": "vn"}}
	if err := store.Put(ctx, profile); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Get(ctx, "default")
	if err != nil || loaded.DisplayName != "Default" || loaded.Metadata["region"] != "vn" {
		t.Fatalf("unexpected profile: %#v, %v", loaded, err)
	}
	loaded.Metadata["region"] = "changed"
	again, err := store.Get(ctx, "default")
	if err != nil || again.Metadata["region"] != "vn" {
		t.Fatal("store leaked mutable metadata")
	}
	if err := store.Delete(ctx, "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "default"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("profile store permissions are not private: %v %v", info, err)
	}
}

func TestFileProfileStoreRejectsEmptyName(t *testing.T) {
	store := NewFileProfileStore(filepath.Join(t.TempDir(), "profiles.json"))
	if err := store.Put(context.Background(), Profile{}); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected invalid profile, got %v", err)
	}
}
