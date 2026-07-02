package parsecontent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// MarkdownDocument contains Markdown body content and parsed frontmatter.
type MarkdownDocument struct {
	Content     string         `json:"content"`
	Frontmatter map[string]any `json:"frontmatter"`
}

// JSON parses one JSON value.
func JSON(content string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%w: decode json: %w", ErrInvalid, err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("%w: multiple json values are not supported", ErrInvalid)
		}

		return nil, fmt.Errorf("%w: decode json: %w", ErrInvalid, err)
	}

	return normalizeValue(value)
}

// YAML parses one YAML document.
func YAML(content string) (any, error) {
	decoder := yaml.NewDecoder(strings.NewReader(content))

	var value any
	if err := decoder.Decode(&value); err != nil {
		if errors.Is(err, io.EOF) {
			return map[string]any{}, nil
		}

		return nil, fmt.Errorf("%w: decode yaml: %w", ErrInvalid, err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("%w: multiple yaml documents are not supported", ErrInvalid)
		}

		return nil, fmt.Errorf("%w: decode yaml: %w", ErrInvalid, err)
	}

	return normalizeValue(value)
}

// TOML parses one TOML document.
func TOML(content string) (any, error) {
	value := map[string]any{}
	if _, err := toml.Decode(content, &value); err != nil {
		return nil, fmt.Errorf("%w: decode toml: %w", ErrInvalid, err)
	}

	return normalizeValue(value)
}

// Markdown parses optional YAML or TOML frontmatter from Markdown text.
func Markdown(content string) (MarkdownDocument, error) {
	delimiter, frontmatter, body, found, err := splitFrontmatter(content)
	if err != nil {
		return MarkdownDocument{}, err
	}
	if !found {
		return MarkdownDocument{Content: content, Frontmatter: map[string]any{}}, nil
	}

	var value any
	switch delimiter {
	case "---":
		value, err = YAML(frontmatter)
	case "+++":
		value, err = TOML(frontmatter)
	default:
		err = fmt.Errorf("%w: unsupported frontmatter delimiter %q", ErrInvalid, delimiter)
	}
	if err != nil {
		return MarkdownDocument{}, err
	}

	frontmatterObject, ok := value.(map[string]any)
	if !ok || frontmatterObject == nil {
		return MarkdownDocument{}, fmt.Errorf("%w: frontmatter must be an object", ErrInvalid)
	}

	return MarkdownDocument{
		Content:     trimLeadingBlankLine(body),
		Frontmatter: frontmatterObject,
	}, nil
}

// MarkdownMap parses Markdown and returns a map for JavaScript and templates.
func MarkdownMap(content string) (map[string]any, error) {
	document, err := Markdown(content)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"content":     document.Content,
		"frontmatter": document.Frontmatter,
	}, nil
}

func splitFrontmatter(content string) (string, string, string, bool, error) {
	firstLine, rest := nextLine(content)
	if firstLine != "---" && firstLine != "+++" {
		return "", "", content, false, nil
	}

	delimiter := firstLine
	frontmatterStart := len(content) - len(rest)
	remaining := rest
	for len(remaining) > 0 {
		line, next := nextLine(remaining)
		lineStart := len(content) - len(remaining)
		if line == delimiter {
			return delimiter, content[frontmatterStart:lineStart], next, true, nil
		}

		remaining = next
	}

	return "", "", "", false, fmt.Errorf("%w: unterminated frontmatter block", ErrInvalid)
}

func nextLine(content string) (string, string) {
	line, rest, ok := strings.Cut(content, "\n")
	if !ok {
		return strings.TrimSuffix(content, "\r"), ""
	}

	return strings.TrimSuffix(line, "\r"), rest
}

func trimLeadingBlankLine(content string) string {
	if trimmed, ok := strings.CutPrefix(content, "\r\n"); ok {
		return trimmed
	}

	trimmed, _ := strings.CutPrefix(content, "\n")
	return trimmed
}

func normalizeValue(value any) (any, error) {
	return normalizeValueAt(value, "$")
}

func normalizeValueAt(value any, location string) (any, error) {
	switch typedValue := value.(type) {
	case nil:
		return nil, nil
	case bool, string:
		return typedValue, nil
	case json.Number:
		return normalizeNumber(typedValue, location)
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

	return normalizeReflectedValue(reflect.ValueOf(value), location)
}

func normalizeNumber(number json.Number, location string) (any, error) {
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

func normalizeFloat(value float64, location string) (float64, error) {
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, fmt.Errorf("%w: %s has non-finite number", ErrValueUnsupported, location)
	}

	return value, nil
}

func normalizeReflectedValue(value reflect.Value, location string) (any, error) {
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

		item, err := normalizeValueAt(iterator.Value().Interface(), location+"."+key)
		if err != nil {
			return nil, err
		}

		items[key] = item
	}

	return items, nil
}

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

func normalizeSlice(value reflect.Value, location string) ([]any, error) {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return nil, nil
	}

	items := make([]any, value.Len())
	for index := range value.Len() {
		item, err := normalizeValueAt(
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
