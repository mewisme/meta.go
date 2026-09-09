package fsutil

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type failingReader struct{ sent bool }

func (r *failingReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, errors.New("injected read failure")
	}
	r.sent = true
	return copy(p, []byte("partial")), nil
}

func TestAtomicWriteReaderPreservesTargetOnReadFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.bin")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := AtomicWriteReader(path, &failingReader{}, 0o600); err == nil {
		t.Fatal("expected injected read failure")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "original" {
		t.Fatalf("target changed after failed write: %q %v", got, err)
	}
}

func TestAtomicWriteReaderReplacesSymlinkInsteadOfFollowingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated privileges on Windows")
	}
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim")
	path := filepath.Join(dir, "output")
	if err := os.WriteFile(victim, []byte("victim"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, path); err != nil {
		t.Fatal(err)
	}
	if _, err := AtomicWriteReader(path, bytes.NewBufferString("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	victimData, _ := os.ReadFile(victim)
	outputData, _ := os.ReadFile(path)
	if string(victimData) != "victim" || string(outputData) != "safe" {
		t.Fatalf("unexpected symlink replacement: victim=%q output=%q", victimData, outputData)
	}
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("output remained a symlink: %v %v", info, err)
	}
}
