package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"

	fberrors "go.mewis.me/meta.go/errors"
)

func applyJQ(selector string, value any) ([]any, error) {
	code, err := compileJQ(selector)
	if err != nil {
		return nil, err
	}
	value, err = normalizeJQValue(value)
	if err != nil {
		return nil, err
	}
	iter := code.Run(value)
	values := make([]any, 0, 1)
	for {
		value, ok := iter.Next()
		if !ok {
			return values, nil
		}
		if err, ok := value.(error); ok {
			return nil, fmt.Errorf("%w: jq expression failed: %v", fberrors.ErrInvalidInput, err)
		}
		values = append(values, value)
	}
}

func normalizeJQValue(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: encode jq input: %v", fberrors.ErrInvalidInput, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var normalized any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, fmt.Errorf("%w: decode jq input: %v", fberrors.ErrInvalidInput, err)
	}
	return normalized, nil
}

func compileJQ(selector string) (*gojq.Code, error) {
	query, err := gojq.Parse(selector)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid jq expression: %v", fberrors.ErrInvalidInput, err)
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, fmt.Errorf("%w: compile jq expression: %v", fberrors.ErrInvalidInput, err)
	}
	return code, nil
}

func writeJSONOutput(out io.Writer, value any, selector string) error {
	values := []any{value}
	if selector != "" {
		var err error
		values, err = applyJQ(selector, value)
		if err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	for _, value := range values {
		if err := encoder.Encode(value); err != nil {
			return err
		}
	}
	return nil
}
