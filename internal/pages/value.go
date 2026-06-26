package pages

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// normalizeJSONValue converts a value into the same shape produced by decoding
// JSON with UseNumber.
func normalizeJSONValue(value any) (any, error) {
	content, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: encode value: %w", ErrValueUnsupported, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	var normalizedValue any
	if err := decoder.Decode(&normalizedValue); err != nil {
		return nil, fmt.Errorf("%w: decode value: %w", ErrValueUnsupported, err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("%w: multiple values", ErrValueUnsupported)
		}

		return nil, fmt.Errorf("%w: decode value: %w", ErrValueUnsupported, err)
	}

	return convertJSONNumbers(normalizedValue)
}

// convertJSONNumbers converts decoded JSON numbers into int64 when possible and
// float64 otherwise.
func convertJSONNumbers(value any) (any, error) {
	switch typedValue := value.(type) {
	case nil, bool, string:
		return typedValue, nil
	case json.Number:
		return convertJSONNumber(typedValue)
	case []any:
		items := make([]any, len(typedValue))
		for index, item := range typedValue {
			convertedItem, err := convertJSONNumbers(item)
			if err != nil {
				return nil, err
			}

			items[index] = convertedItem
		}

		return items, nil
	case map[string]any:
		items := make(map[string]any, len(typedValue))
		for key, item := range typedValue {
			convertedItem, err := convertJSONNumbers(item)
			if err != nil {
				return nil, err
			}

			items[key] = convertedItem
		}

		return items, nil
	default:
		return nil, fmt.Errorf("%w: unsupported decoded value %T", ErrValueUnsupported, value)
	}
}

// convertJSONNumber converts one decoded JSON number to an integer or float.
func convertJSONNumber(number json.Number) (any, error) {
	if integer, err := number.Int64(); err == nil {
		return integer, nil
	}

	float, err := number.Float64()
	if err != nil {
		return nil, fmt.Errorf("%w: invalid number %q", ErrValueUnsupported, number)
	}

	return float, nil
}
