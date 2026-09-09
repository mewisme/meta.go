package storage

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("storage entry not found")

type Profile struct {
	Name        string            `json:"name" yaml:"name" toml:"name"`
	DisplayName string            `json:"display_name,omitempty" yaml:"display_name,omitempty" toml:"display_name,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty" toml:"metadata,omitempty"`
}

type ProfileStore interface {
	Get(context.Context, string) (Profile, error)
	Put(context.Context, Profile) error
	Delete(context.Context, string) error
	List(context.Context) ([]Profile, error)
}

type SecretStore interface {
	Get(context.Context, string, string) ([]byte, error)
	Put(context.Context, string, string, []byte) error
	Delete(context.Context, string, string) error
}
