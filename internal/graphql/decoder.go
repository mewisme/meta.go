package graphql

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func DecodeBatch(data []byte) ([]json.RawMessage, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4*1024), 16*1024*1024)
	var result []json.RawMessage
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("for (;;);")) {
			line = bytes.TrimSpace(line[len("for (;;);"):])
		}
		if !json.Valid(line) {
			return nil, errors.New("invalid GraphQL batch JSON")
		}
		if len(line) == 0 || line[0] != '{' {
			return nil, errors.New("GraphQL batch item is not an object")
		}
		result = append(result, append(json.RawMessage(nil), line...))
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return result, nil
}
