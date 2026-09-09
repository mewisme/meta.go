package webapi

import (
	"bytes"
	"io"
	"testing"
)

func BenchmarkBoundedMediaStreaming(b *testing.B) {
	payload := bytes.Repeat([]byte("x"), 1<<20)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for b.Loop() {
		body := io.NopCloser(bytes.NewReader(payload))
		reader := &limitReadCloser{reader: io.LimitReader(body, int64(len(payload))+1), closer: body, remaining: int64(len(payload))}
		if n, err := io.Copy(io.Discard, reader); err != nil || n != int64(len(payload)) {
			b.Fatalf("stream failed: %d %v", n, err)
		}
	}
}
