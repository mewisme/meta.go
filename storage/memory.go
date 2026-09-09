package storage

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type MemoryProfileStore struct {
	mu       sync.RWMutex
	profiles map[string]Profile
}

func NewMemoryProfileStore() *MemoryProfileStore {
	return &MemoryProfileStore{profiles: map[string]Profile{}}
}

func (s *MemoryProfileStore) Get(ctx context.Context, name string) (Profile, error) {
	if err := ctx.Err(); err != nil {
		return Profile{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.profiles[name]
	if !ok {
		return Profile{}, ErrNotFound
	}
	return cloneProfile(p), nil
}

func (s *MemoryProfileStore) Put(ctx context.Context, profile Profile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	profile.Name = strings.TrimSpace(profile.Name)
	if profile.Name == "" || strings.ContainsRune(profile.Name, '\x00') {
		return ErrInvalidProfile
	}
	s.mu.Lock()
	s.profiles[profile.Name] = cloneProfile(profile)
	s.mu.Unlock()
	return nil
}

func (s *MemoryProfileStore) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.profiles[name]; !ok {
		return ErrNotFound
	}
	delete(s.profiles, name)
	return nil
}

func (s *MemoryProfileStore) List(ctx context.Context) ([]Profile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	result := make([]Profile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		result = append(result, cloneProfile(profile))
	}
	s.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

type MemorySecretStore struct {
	mu      sync.RWMutex
	secrets map[string][]byte
}

func NewMemorySecretStore() *MemorySecretStore {
	return &MemorySecretStore{secrets: map[string][]byte{}}
}

func secretKey(profile, key string) string { return profile + "\x00" + key }

func (s *MemorySecretStore) Get(ctx context.Context, profile, key string) ([]byte, error) {
	if err := validateSecretKey(profile, key); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	value, ok := s.secrets[secretKey(profile, key)]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte(nil), value...), nil
}

func (s *MemorySecretStore) Put(ctx context.Context, profile, key string, value []byte) error {
	if err := validateSecretKey(profile, key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	s.secrets[secretKey(profile, key)] = append([]byte(nil), value...)
	s.mu.Unlock()
	return nil
}

func (s *MemorySecretStore) Delete(ctx context.Context, profile, key string) error {
	if err := validateSecretKey(profile, key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := secretKey(profile, key)
	if _, ok := s.secrets[k]; !ok {
		return ErrNotFound
	}
	delete(s.secrets, k)
	return nil
}
