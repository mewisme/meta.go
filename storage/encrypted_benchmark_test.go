package storage

import (
	"bytes"
	"context"
	"testing"
)

func BenchmarkEncryptedStateLoadSave(b *testing.B) {
	backend := NewMemorySecretStore()
	store, err := NewEncryptedSecretStore(backend, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		b.Fatal(err)
	}
	payload := bytes.Repeat([]byte("state"), 16<<10)
	ctx := context.Background()
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for b.Loop() {
		if err := store.Put(ctx, "bench", "e2ee_state", payload); err != nil {
			b.Fatal(err)
		}
		got, err := store.Get(ctx, "bench", "e2ee_state")
		if err != nil || len(got) != len(payload) {
			b.Fatalf("load failed: %d %v", len(got), err)
		}
	}
}
