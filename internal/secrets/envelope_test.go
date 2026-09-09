package secrets

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	aad := []byte("profile:default/state")
	plaintext := []byte("sensitive state")
	envelope, err := Seal(key, aad, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(key, aad, envelope)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(opened, plaintext) {
		t.Fatalf("unexpected plaintext: %q", opened)
	}
}

func TestEnvelopeRejectsTamperingAndWrongAAD(t *testing.T) {
	key := make([]byte, chacha20poly1305.KeySize)
	envelope, err := Seal(key, []byte("aad"), []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), envelope...)
	tampered[len(tampered)-1] ^= 1
	if _, err := Open(key, []byte("aad"), tampered); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("expected tamper rejection, got %v", err)
	}
	if _, err := Open(key, []byte("wrong"), envelope); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("expected AAD rejection, got %v", err)
	}
}

func TestEnvelopeRejectsInvalidKey(t *testing.T) {
	if _, err := Seal([]byte("short"), nil, nil); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected invalid key, got %v", err)
	}
}
