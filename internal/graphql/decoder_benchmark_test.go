package graphql

import (
	"bytes"
	"testing"
)

func BenchmarkDecodeBatch(b *testing.B) {
	payload := bytes.Repeat([]byte(`{"data":{"id":"1","name":"example"}}`+"\n"), 100)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for b.Loop() {
		items, err := DecodeBatch(payload)
		if err != nil || len(items) != 100 {
			b.Fatalf("decode failed: %d %v", len(items), err)
		}
	}
}
