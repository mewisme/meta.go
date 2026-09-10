package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"

	fberrors "go.mewis.me/meta.go/errors"
)

const (
	maxMediaChunkBytes = 64 << 10
	maxMediaBytes      = 100 << 20
)

type mediaResult[T any] struct {
	value T
	err   error
}

func receiveMedia[T any](ctx context.Context, size int64, expectedSHA256 []byte, recv func() ([]byte, error), consume func(io.Reader) (T, error)) (T, []byte, error) {
	var zero T
	if size < 0 || size > maxMediaBytes {
		return zero, nil, fmt.Errorf("%w: media size must be between 0 and %d bytes", fberrors.ErrInvalidInput, maxMediaBytes)
	}
	if len(expectedSHA256) != 0 && len(expectedSHA256) != sha256.Size {
		return zero, nil, fmt.Errorf("%w: sha256 must be %d bytes", fberrors.ErrInvalidInput, sha256.Size)
	}
	reader, writer := io.Pipe()
	resultCh := make(chan mediaResult[T], 1)
	go func() {
		value, err := consume(reader)
		_ = reader.CloseWithError(err)
		resultCh <- mediaResult[T]{value: value, err: err}
	}()
	stop := context.AfterFunc(ctx, func() { _ = writer.CloseWithError(ctx.Err()) })
	defer stop()
	hash := sha256.New()
	var received int64
	for {
		data, err := recv()
		if err == io.EOF {
			actual := hash.Sum(nil)
			var validationErr error
			switch {
			case received != size:
				validationErr = fmt.Errorf("%w: media size mismatch: declared %d, received %d", fberrors.ErrInvalidInput, size, received)
			case len(expectedSHA256) > 0 && !bytes.Equal(actual, expectedSHA256):
				validationErr = fmt.Errorf("%w: media sha256 mismatch", fberrors.ErrInvalidInput)
			}
			if validationErr != nil {
				_ = writer.CloseWithError(validationErr)
			} else {
				_ = writer.Close()
			}
			result := <-resultCh
			if validationErr != nil {
				return zero, actual, validationErr
			}
			if result.err != nil {
				return zero, actual, result.err
			}
			return result.value, actual, nil
		}
		if err != nil {
			_ = writer.CloseWithError(err)
			result := <-resultCh
			if result.err != nil {
				return zero, nil, result.err
			}
			return zero, nil, err
		}
		if len(data) > maxMediaChunkBytes {
			err := fmt.Errorf("%w: media chunk exceeds %d bytes", fberrors.ErrInvalidInput, maxMediaChunkBytes)
			_ = writer.CloseWithError(err)
			<-resultCh
			return zero, nil, err
		}
		if received+int64(len(data)) > size {
			err := fmt.Errorf("%w: media exceeds declared size %d", fberrors.ErrInvalidInput, size)
			_ = writer.CloseWithError(err)
			<-resultCh
			return zero, nil, err
		}
		if len(data) == 0 {
			continue
		}
		received += int64(len(data))
		_, _ = hash.Write(data)
		if _, err := writer.Write(data); err != nil {
			result := <-resultCh
			if result.err != nil {
				return zero, nil, result.err
			}
			return zero, nil, err
		}
	}
}
