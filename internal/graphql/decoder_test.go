package graphql

import "testing"

func TestDecodeBatch(t *testing.T) {
	items, err := DecodeBatch([]byte("for (;;);{\"a\":1}\n{\"b\":2}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
}

func FuzzDecodeBatch(f *testing.F) {
	f.Add([]byte("for (;;);{\"a\":1}\n"))
	f.Add([]byte("{}\n{}\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeBatch(data)
	})
}
