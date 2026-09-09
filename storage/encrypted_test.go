package storage

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestEncryptedSecretStoreRoundTrip(t *testing.T) {
	backend := NewMemorySecretStore()
	key := bytes.Repeat([]byte{1}, 32)
	store, err := NewEncryptedSecretStore(backend, key)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	plain := []byte("secret device state")
	if err := store.Put(ctx, "default", "e2ee_state", plain); err != nil {
		t.Fatal(err)
	}
	raw, err := backend.Get(ctx, "default", "e2ee_state")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, plain) {
		t.Fatal("plaintext leaked into backend store")
	}
	loaded, err := store.Get(ctx, "default", "e2ee_state")
	if err != nil || !bytes.Equal(loaded, plain) {
		t.Fatalf("unexpected decrypted state: %q %v", loaded, err)
	}
}

func TestEncryptedSecretStoreAADIsolation(t *testing.T) {
	backend := NewMemorySecretStore()
	store, _ := NewEncryptedSecretStore(backend, bytes.Repeat([]byte{2}, 32))
	ctx := context.Background()
	if err := store.Put(ctx, "a", "state", []byte("x")); err != nil {
		t.Fatal(err)
	}
	raw, _ := backend.Get(ctx, "a", "state")
	_ = backend.Put(ctx, "b", "state", raw)
	if _, err := store.Get(ctx, "b", "state"); err == nil {
		t.Fatal("expected AAD authentication failure")
	}
}

func TestEncryptedSecretStoreRejectsInvalidKey(t *testing.T) {
	_, err := NewEncryptedSecretStore(NewMemorySecretStore(), []byte("short"))
	if !errors.Is(err, ErrInvalidEncryptionKey) {
		t.Fatalf("unexpected error: %v", err)
	}
}
