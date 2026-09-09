package cli

import (
	"errors"
	"testing"

	fberrors "go.mewis.me/fbgo/errors"
)

func TestDecodeJSONInputRequiresSingleJQValue(t *testing.T) {
	var target map[string]string
	for _, selector := range []string{".missing | empty", ".items[]"} {
		data := []byte(`{"items":[{"id":"1"},{"id":"2"}]}`)
		if err := decodeJSONInput(data, selector, &target); !errors.Is(err, fberrors.ErrInvalidInput) {
			t.Fatalf("selector %q: expected invalid input, got %v", selector, err)
		}
	}
}

func TestDecodeJSONInputRejectsInvalidSelector(t *testing.T) {
	var target map[string]string
	if err := decodeJSONInput([]byte(`{"id":"1"}`), ".[", &target); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
