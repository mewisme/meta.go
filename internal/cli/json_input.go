package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	fberrors "go.mewis.me/meta.go/errors"
)

func decodeJSONInput(data []byte, selector string, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("%w: invalid JSON input: %v", fberrors.ErrInvalidInput, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("%w: JSON input must contain exactly one value", fberrors.ErrInvalidInput)
		}
		return fmt.Errorf("%w: invalid JSON input: %v", fberrors.ErrInvalidInput, err)
	}
	if selector != "" {
		values, err := applyJQ(selector, value)
		if err != nil {
			return err
		}
		if len(values) == 0 {
			return fmt.Errorf("%w: jq expression returned no values", fberrors.ErrInvalidInput)
		}
		if len(values) != 1 {
			return fmt.Errorf("%w: jq expression must return exactly one value", fberrors.ErrInvalidInput)
		}
		value = values[0]
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: encode selected JSON value: %v", fberrors.ErrInvalidInput, err)
	}
	strict := json.NewDecoder(bytes.NewReader(encoded))
	strict.DisallowUnknownFields()
	if err := strict.Decode(target); err != nil {
		return fmt.Errorf("%w: decode JSON input: %v", fberrors.ErrInvalidInput, err)
	}
	return nil
}
