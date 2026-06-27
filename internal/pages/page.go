package pages

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
)

// decodePage converts one generator item into a normalized Page.
func decodePage(generator string, index int, value any) (Page, error) {
	normalizedValue, err := normalizeJSONValue(value)
	if err != nil {
		return Page{}, fmt.Errorf("%w: %s[%d]: %w", ErrPageInvalid, generator, index, err)
	}

	object, ok := objectFields(normalizedValue)
	if !ok {
		return Page{}, pageError(generator, index, "must be an object")
	}

	permalink, err := requiredStringField(generator, index, object, "permalink")
	if err != nil {
		return Page{}, err
	}
	normalizedPermalink, outputPath, err := normalizePermalink(permalink)
	if err != nil {
		return Page{}, fmt.Errorf("%w: %s[%d].permalink: %w", ErrPageInvalid, generator, index, err)
	}

	layout, err := requiredNonEmptyStringField(generator, index, object, "layout")
	if err != nil {
		return Page{}, err
	}
	if _, err := requiredStringField(generator, index, object, "content"); err != nil {
		return Page{}, err
	}

	fields := clonePageFields(object)
	fields["generator"] = generator
	fields["index"] = int64(index)
	fields["layout"] = layout
	fields["outputPath"] = outputPath
	fields["permalink"] = normalizedPermalink

	return Page{
		Fields:     fields,
		Generator:  generator,
		Index:      index,
		Layout:     layout,
		OutputPath: outputPath,
		Permalink:  normalizedPermalink,
	}, nil
}

// clonePageFields returns a writable copy of a normalized page object.
func clonePageFields(fields map[string]any) map[string]any {
	clone := make(map[string]any, len(fields)+4)
	maps.Copy(clone, fields)

	return clone
}

// objectFields converts any map with string keys into map[string]any.
func objectFields(value any) (map[string]any, bool) {
	reflectedValue := reflect.ValueOf(value)
	if !reflectedValue.IsValid() {
		return nil, false
	}

	for reflectedValue.Kind() == reflect.Interface || reflectedValue.Kind() == reflect.Pointer {
		if reflectedValue.IsNil() {
			return nil, false
		}

		reflectedValue = reflectedValue.Elem()
	}

	if reflectedValue.Kind() != reflect.Map {
		return nil, false
	}

	fields := make(map[string]any, reflectedValue.Len())
	iterator := reflectedValue.MapRange()
	for iterator.Next() {
		key, ok := reflectedStringKey(iterator.Key())
		if !ok {
			return nil, false
		}

		fields[key] = iterator.Value().Interface()
	}

	return fields, true
}

// reflectedStringKey extracts a string from a reflected map key.
func reflectedStringKey(value reflect.Value) (string, bool) {
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

// requiredStringField returns a required page string field.
func requiredStringField(
	generator string,
	index int,
	object map[string]any,
	field string,
) (string, error) {
	value, ok := object[field]
	if !ok {
		return "", pageError(generator, index, "%s is required", field)
	}

	stringValue, ok := value.(string)
	if !ok {
		return "", pageError(generator, index, "%s must be a string", field)
	}

	return stringValue, nil
}

// requiredNonEmptyStringField returns a required non-empty page string field.
func requiredNonEmptyStringField(
	generator string,
	index int,
	object map[string]any,
	field string,
) (string, error) {
	value, err := requiredStringField(generator, index, object, field)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", pageError(generator, index, "%s cannot be empty", field)
	}

	return value, nil
}

// pageError returns a contextual page contract error.
func pageError(generator string, index int, format string, args ...any) error {
	return fmt.Errorf(
		"%w: %s[%d]: %s",
		ErrPageInvalid,
		generator,
		index,
		fmt.Sprintf(format, args...),
	)
}
