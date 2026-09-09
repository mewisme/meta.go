package meta

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

func TestMemoryDeviceStoreState(t *testing.T) {
	store, err := NewMemoryDeviceStore()
	if err != nil {
		t.Fatal(err)
	}
	if store.Device() == nil || store.Device().IdentityKey == nil || store.Device().RegistrationID == 0 {
		t.Fatal("device key material was not initialized")
	}
	ctx := context.Background()
	session := []byte("signal-session")
	if err := store.PutSession(ctx, "123:1", session); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetSession(ctx, "123:1")
	if err != nil || !bytes.Equal(loaded, session) {
		t.Fatalf("unexpected session: %q %v", loaded, err)
	}
	loaded[0] ^= 1
	again, _ := store.GetSession(ctx, "123:1")
	if bytes.Equal(loaded, again) {
		t.Fatal("session getter leaked mutable storage")
	}
	preKeys, err := store.GetOrGenPreKeys(ctx, 3)
	if err != nil || len(preKeys) != 3 {
		t.Fatalf("unexpected prekeys: %d %v", len(preKeys), err)
	}
	if err := store.MarkPreKeysAsUploaded(ctx, preKeys[1].KeyID); err != nil {
		t.Fatal(err)
	}
	count, err := store.UploadedPreKeyCount(ctx)
	if err != nil || count != 2 {
		t.Fatalf("unexpected uploaded prekey count: %d %v", count, err)
	}
}

func TestDeviceGenerationPropagatesRandomFailure(t *testing.T) {
	_, err := newMemoryDeviceStore(failingReader{})
	if err == nil {
		t.Fatal("expected random source failure")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestDeviceStoreContextCancellation(t *testing.T) {
	store, err := NewMemoryDeviceStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.PutSession(ctx, "1:1", []byte("x")); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
