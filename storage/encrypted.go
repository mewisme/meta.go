package storage

import (
	"context"
	"errors"

	internalsecrets "go.mewis.me/meta.go/internal/secrets"
)

var ErrInvalidEncryptionKey = errors.New("invalid secret-store encryption key")

type EncryptedSecretStore struct {
	backend SecretStore
	key     []byte
}

func NewEncryptedSecretStore(backend SecretStore, key []byte) (*EncryptedSecretStore, error) {
	if backend == nil || len(key) != 32 {
		return nil, ErrInvalidEncryptionKey
	}
	return &EncryptedSecretStore{backend: backend, key: append([]byte(nil), key...)}, nil
}

func (s *EncryptedSecretStore) Get(ctx context.Context, profile, name string) ([]byte, error) {
	data, err := s.backend.Get(ctx, profile, name)
	if err != nil {
		return nil, err
	}
	return internalsecrets.Open(s.key, secretAAD(profile, name), data)
}

func (s *EncryptedSecretStore) Put(ctx context.Context, profile, name string, value []byte) error {
	sealed, err := internalsecrets.Seal(s.key, secretAAD(profile, name), value)
	if err != nil {
		return err
	}
	return s.backend.Put(ctx, profile, name, sealed)
}

func (s *EncryptedSecretStore) Delete(ctx context.Context, profile, name string) error {
	return s.backend.Delete(ctx, profile, name)
}

func secretAAD(profile, name string) []byte { return []byte("meta:" + profile + ":" + name) }
