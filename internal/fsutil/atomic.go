package fsutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func EnsurePrivateDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return os.Chmod(path, 0o700)
}

func AtomicWriteFile(path string, data []byte, mode os.FileMode) error {
	_, err := AtomicWriteReader(path, bytes.NewReader(data), mode)
	return err
}

func AtomicWriteReader(path string, reader io.Reader, mode os.FileMode) (int64, error) {
	if reader == nil {
		return 0, errors.New("nil reader")
	}
	dir := filepath.Dir(path)
	if err := EnsurePrivateDir(dir); err != nil {
		return 0, fmt.Errorf("create private directory: %w", err)
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return 0, fmt.Errorf("create temporary file: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return 0, err
	}
	written, err := io.Copy(f, reader)
	if err != nil {
		f.Close()
		return written, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return written, err
	}
	if err := f.Close(); err != nil {
		return written, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return written, err
	}
	if d, err := os.Open(dir); err == nil {
		defer d.Close()
		_ = d.Sync()
	}
	return written, nil
}

func ExclusiveWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := EnsurePrivateDir(dir); err != nil {
		return fmt.Errorf("create private directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(path)
		}
	}()
	if n, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	} else if n != len(data) {
		_ = f.Close()
		return io.ErrShortWrite
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	remove = false
	if d, err := os.Open(dir); err == nil {
		defer d.Close()
		_ = d.Sync()
	}
	return nil
}
