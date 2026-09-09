package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCanonicalConfigPathStableAcrossSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(real, []byte("schema_version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	want, err := canonicalConfigPath(real)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	link := filepath.Join(dir, "config-link.toml")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	got, err := canonicalConfigPath(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("symlink config path changed key identity: %q != %q", got, want)
	}
}

func TestCanonicalConfigPathAllowsMissingTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "config.toml")
	got, err := canonicalConfigPath(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("canonical path = %q, want %q", got, filepath.Clean(want))
	}
}
