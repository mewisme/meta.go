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
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var result []json.RawMessage
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("for (;;);")) {
			line = bytes.TrimSpace(line[len("for (;;);"):])
		}
		var raw json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 || raw[0] != '{' {
			return nil, errors.New("GraphQL batch item is not an object")
		}
		result = append(result, append(json.RawMessage(nil), raw...))
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return result, nil
}
