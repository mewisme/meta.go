package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"

	"go.mewis.me/fbgo/internal/fsutil"
)

var ErrInvalidProfile = errors.New("invalid profile")

type FileProfileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileProfileStore(path string) *FileProfileStore { return &FileProfileStore{path: path} }

func (s *FileProfileStore) Get(ctx context.Context, name string) (Profile, error) {
	if err := ctx.Err(); err != nil {
		return Profile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.load()
	if err != nil {
		return Profile{}, err
	}
	profile, ok := profiles[name]
	if !ok {
		return Profile{}, ErrNotFound
	}
	return cloneProfile(profile), nil
}

func (s *FileProfileStore) Put(ctx context.Context, profile Profile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	profile.Name = strings.TrimSpace(profile.Name)
	if profile.Name == "" || strings.ContainsRune(profile.Name, '\x00') {
		return ErrInvalidProfile
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.load()
	if err != nil {
		return err
	}
	profiles[profile.Name] = cloneProfile(profile)
	return s.save(profiles)
}

func (s *FileProfileStore) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := profiles[name]; !ok {
		return ErrNotFound
	}
	delete(profiles, name)
	return s.save(profiles)
}

func (s *FileProfileStore) List(ctx context.Context) ([]Profile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.load()
	if err != nil {
		return nil, err
	}
	result := make([]Profile, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, cloneProfile(profile))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (s *FileProfileStore) load() (map[string]Profile, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]Profile{}, nil
	}
	if err != nil {
		return nil, err
	}
	profiles := map[string]Profile{}
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *FileProfileStore) save(profiles map[string]Profile) error {
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return fsutil.AtomicWriteFile(s.path, data, 0o600)
}

func cloneProfile(profile Profile) Profile {
	if profile.Metadata == nil {
		return profile
	}
	metadata := make(map[string]string, len(profile.Metadata))
	for key, value := range profile.Metadata {
		metadata[key] = value
	}
	profile.Metadata = metadata
	return profile
}
