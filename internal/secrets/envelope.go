package secrets

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const EnvelopeVersion byte = 1

var (
	ErrInvalidKey      = errors.New("invalid encryption key")
	ErrInvalidEnvelope = errors.New("invalid encrypted envelope")
)

// Seal encrypts plaintext using XChaCha20-Poly1305 and binds optional AAD.
func Seal(key, aad, plaintext []byte) ([]byte, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, ErrInvalidKey
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	result := make([]byte, 1+len(nonce), 1+len(nonce)+len(plaintext)+aead.Overhead())
	result[0] = EnvelopeVersion
	copy(result[1:], nonce)
	result = aead.Seal(result, nonce, plaintext, aad)
	return result, nil
}

// Open decrypts and authenticates an envelope created by Seal.
func Open(key, aad, envelope []byte) ([]byte, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, ErrInvalidKey
	}
	minimum := 1 + chacha20poly1305.NonceSizeX + chacha20poly1305.Overhead
	if len(envelope) < minimum || envelope[0] != EnvelopeVersion {
		return nil, ErrInvalidEnvelope
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce := envelope[1 : 1+chacha20poly1305.NonceSizeX]
	plaintext, err := aead.Open(nil, nonce, envelope[1+chacha20poly1305.NonceSizeX:], aad)
	if err != nil {
		return nil, fmt.Errorf("%w: authentication failed", ErrInvalidEnvelope)
	}
	return plaintext, nil
}
