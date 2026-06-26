package data

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"time"
)

// normalizeJSONValue converts supported parser and JavaScript values into a
// predictable JSON-compatible shape.
func normalizeJSONValue(value any) (any, error) {
	return normalizeJSONValueAt(value, "$")
}

// normalizeJSONValueAt normalizes a value while preserving its location for
// useful errors.
func normalizeJSONValueAt(value any, location string) (any, error) {
	switch typedValue := value.(type) {
	case nil:
		return nil, nil
	case bool, string:
		return typedValue, nil
	case json.Number:
		return normalizeJSONNumber(typedValue, location)
	case int:
		return int64(typedValue), nil
	case int8:
		return int64(typedValue), nil
	case int16:
		return int64(typedValue), nil
	case int32:
		return int64(typedValue), nil
	case int64:
		return typedValue, nil
	case uint:
		return uint64(typedValue), nil
	case uint8:
		return uint64(typedValue), nil
	case uint16:
		return uint64(typedValue), nil
	case uint32:
		return uint64(typedValue), nil
	case uint64:
		return typedValue, nil
	case float32:
		return normalizeFloat(float64(typedValue), location)
	case float64:
		return normalizeFloat(typedValue, location)
	case time.Time:
		return typedValue.Format(time.RFC3339Nano), nil
	}

	return normalizeReflectedJSONValue(reflect.ValueOf(value), location)
}

// normalizeJSONNumber converts a decoded JSON number to an integer when possible
// and otherwise to a finite float.
func normalizeJSONNumber(number json.Number, location string) (any, error) {
	if integer, err := number.Int64(); err == nil {
		return integer, nil
	}

	float, err := number.Float64()
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %s has invalid number %q",
			ErrValueUnsupported,
			location,
			number,
		)
	}

	return normalizeFloat(float, location)
}

// normalizeFloat rejects numbers that JSON cannot represent.
func normalizeFloat(value float64, location string) (float64, error) {
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, fmt.Errorf("%w: %s has non-finite number", ErrValueUnsupported, location)
	}

	return value, nil
}

// normalizeReflectedJSONValue handles map, slice, array, pointer, and interface
// values not covered by direct type switches.
func normalizeReflectedJSONValue(value reflect.Value, location string) (any, error) {
	if !value.IsValid() {
		return nil, nil
	}

	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, nil
		}

		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Map:
		return normalizeMap(value, location)
	case reflect.Slice, reflect.Array:
		return normalizeSlice(value, location)
	default:
		return nil, fmt.Errorf(
			"%w: %s has unsupported value type %s",
			ErrValueUnsupported,
			location,
			value.Type(),
		)
	}
}

// normalizeMap converts any map with string keys into map[string]any.
func normalizeMap(value reflect.Value, location string) (map[string]any, error) {
	if value.IsNil() {
		return nil, nil
	}

	items := make(map[string]any, value.Len())
	iterator := value.MapRange()
	for iterator.Next() {
		key, ok := stringMapKey(iterator.Key())
		if !ok {
			return nil, fmt.Errorf("%w: %s has non-string map key", ErrValueUnsupported, location)
		}

		item, err := normalizeJSONValueAt(iterator.Value().Interface(), location+"."+key)
		if err != nil {
			return nil, err
		}

		items[key] = item
	}

	return items, nil
}

// stringMapKey extracts a JSON object key from a reflected map key.
func stringMapKey(value reflect.Value) (string, bool) {
	for value.Kind() == reflect.Interface {
		if value.IsNil() {
			return "", false
		}

		value = value.Elem()
	}

	if value.Kind() != reflect.String {
		return "", false
	}

	return value.String(), true
}

// normalizeSlice converts reflected arrays and slices into []any.
func normalizeSlice(value reflect.Value, location string) ([]any, error) {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return nil, nil
	}

	items := make([]any, value.Len())
	for index := range value.Len() {
		item, err := normalizeJSONValueAt(
			value.Index(index).Interface(),
			fmt.Sprintf("%s[%d]", location, index),
		)
		if err != nil {
			return nil, err
		}

		items[index] = item
	}

	return items, nil
}
