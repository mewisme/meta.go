package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"

	"go.mewis.me/meta.go/internal/fsutil"
)

type FileSecretStore struct {
	path string
	mu   sync.Mutex
}

func NewFileSecretStore(path string) *FileSecretStore { return &FileSecretStore{path: path} }

func (s *FileSecretStore) Get(ctx context.Context, profile, name string) ([]byte, error) {
	if err := validateSecretKey(profile, name); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return nil, err
	}
	value, ok := entries[secretKey(profile, name)]
	if !ok {
		return nil, ErrNotFound
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func (s *FileSecretStore) Put(ctx context.Context, profile, name string, value []byte) error {
	if err := validateSecretKey(profile, name); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return err
	}
	entries[secretKey(profile, name)] = base64.StdEncoding.EncodeToString(value)
	return s.save(entries)
}

func (s *FileSecretStore) Delete(ctx context.Context, profile, name string) error {
	if err := validateSecretKey(profile, name); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return err
	}
	key := secretKey(profile, name)
	if _, ok := entries[key]; !ok {
		return ErrNotFound
	}
	delete(entries, key)
	return s.save(entries)
}

var ErrInvalidSecretKey = errors.New("invalid secret key")

func validateSecretKey(profile, name string) error {
	if strings.TrimSpace(profile) == "" || strings.TrimSpace(name) == "" || strings.ContainsRune(profile, '\x00') || strings.ContainsRune(name, '\x00') {
		return ErrInvalidSecretKey
	}
	return nil
}

func (s *FileSecretStore) load() (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries := map[string]string{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *FileSecretStore) save(entries map[string]string) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return fsutil.AtomicWriteFile(s.path, data, 0o600)
}
