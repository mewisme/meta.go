package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	fberrors "go.mewis.me/meta.go/errors"
)

func TestReceiveMediaStreamsWithoutWholeFileBuffer(t *testing.T) {
	chunk := bytes.Repeat([]byte{0x5a}, maxMediaChunkBytes)
	const chunks = 128
	declared := int64(len(chunk) * chunks)
	h := sha256.New()
	for range chunks {
		_, _ = h.Write(chunk)
	}
	expected := h.Sum(nil)
	calls := 0
	count, actual, err := receiveMedia(t.Context(), declared, expected, func() ([]byte, error) {
		if calls == chunks {
			return nil, io.EOF
		}
		calls++
		return chunk, nil
	}, func(reader io.Reader) (int64, error) {
		return io.Copy(io.Discard, reader)
	})
	if err != nil || count != declared || !bytes.Equal(actual, expected) || calls != chunks {
		t.Fatalf("unexpected stream result: count=%d calls=%d hash=%x err=%v", count, calls, actual, err)
	}
}

func TestReceiveMediaBackpressure(t *testing.T) {
	chunk := bytes.Repeat([]byte{1}, maxMediaChunkBytes)
	var calls atomic.Int32
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, _, err := receiveMedia(t.Context(), int64(len(chunk)), nil, func() ([]byte, error) {
			if calls.Add(1) == 1 {
				return chunk, nil
			}
			return nil, io.EOF
		}, func(reader io.Reader) (struct{}, error) {
			<-release
			_, err := io.Copy(io.Discard, reader)
			return struct{}{}, err
		})
		done <- err
	}()
	deadline := time.Now().Add(time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(10 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("producer ran ahead of blocked consumer: calls=%d", calls.Load())
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestReceiveMediaValidation(t *testing.T) {
	chunk := []byte("hello")
	wrongHash := bytes.Repeat([]byte{1}, sha256.Size)
	_, _, err := receiveMedia(t.Context(), int64(len(chunk)), wrongHash, chunkReceiver(chunk), discardMedia)
	if !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected checksum validation error, got %v", err)
	}
	_, _, err = receiveMedia(t.Context(), int64(len(chunk)+1), nil, chunkReceiver(chunk), discardMedia)
	if !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected size validation error, got %v", err)
	}
	_, _, err = receiveMedia(t.Context(), int64(maxMediaChunkBytes+1), nil, chunkReceiver(bytes.Repeat([]byte{1}, maxMediaChunkBytes+1)), discardMedia)
	if !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected chunk validation error, got %v", err)
	}
	_, _, err = receiveMedia(t.Context(), maxMediaBytes+1, nil, func() ([]byte, error) { return nil, io.EOF }, discardMedia)
	if !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected total size validation error, got %v", err)
	}
}

func TestReceiveMediaCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		_, _, err := receiveMedia(ctx, int64(maxMediaChunkBytes*2), nil, func() ([]byte, error) {
			if calls.Add(1) == 1 {
				return bytes.Repeat([]byte{1}, maxMediaChunkBytes), nil
			}
			<-ctx.Done()
			return nil, ctx.Err()
		}, discardMedia)
		done <- err
	}()
	deadline := time.Now().Add(time.Second)
	for calls.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after cancellation")
	}
}

func chunkReceiver(chunk []byte) func() ([]byte, error) {
	sent := false
	return func() ([]byte, error) {
		if sent {
			return nil, io.EOF
		}
		sent = true
		return chunk, nil
	}
}

func discardMedia(reader io.Reader) (struct{}, error) {
	_, err := io.Copy(io.Discard, reader)
	return struct{}{}, err
}
