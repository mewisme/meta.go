package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInitDefaultCreatesTOML(t *testing.T) {
	dir := t.TempDir()
	path, err := InitDefault(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != DefaultFileName {
		t.Fatalf("unexpected default filename: %s", path)
	}
	config, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.SchemaVersion != CurrentSchemaVersion || config.TimeoutText != "30s" {
		t.Fatalf("unexpected default config: %#v", config)
	}
	if _, err := InitDefault(dir); !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected exclusive create failure, got %v", err)
	}
}

func TestDiscoverRejectsAmbiguousDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := SaveFile(filepath.Join(dir, "config.toml"), Default()); err != nil {
		t.Fatal(err)
	}
	if err := SaveFile(filepath.Join(dir, "config.json"), Default()); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(dir); !errors.Is(err, ErrAmbiguousConfig) {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

func TestStrictUnknownFields(t *testing.T) {
	for _, ext := range []string{".json", ".yaml", ".toml"} {
		path := filepath.Join(t.TempDir(), "config"+ext)
		var data []byte
		switch ext {
		case ".json":
			data = []byte(`{"schema_version":1,"unknown":true}`)
		case ".yaml":
			data = []byte("schema_version: 1\nunknown: true\n")
		default:
			data = []byte("schema_version = 1\nunknown = true\n")
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadFile(path); err == nil {
			t.Fatalf("expected strict decoding failure for %s", ext)
		}
	}
}
