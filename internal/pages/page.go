package pages

import (
	"fmt"
	"reflect"
)

var allowedPageFields = map[string]struct{}{
	"content":   {},
	"data":      {},
	"date":      {},
	"layout":    {},
	"permalink": {},
	"title":     {},
}

// decodePage converts one generator item into a normalized Page.
func decodePage(generator string, index int, value any) (Page, error) {
	object, ok := objectFields(value)
	if !ok {
		return Page{}, pageError(generator, index, "must be an object")
	}

	if err := rejectUnknownFields(generator, index, object); err != nil {
		return Page{}, err
	}

	permalink, err := requiredStringField(generator, index, object, "permalink")
	if err != nil {
		return Page{}, err
	}
	normalizedPermalink, outputPath, err := normalizePermalink(permalink)
	if err != nil {
		return Page{}, fmt.Errorf("%w: %s[%d].permalink: %w", ErrPageInvalid, generator, index, err)
	}

	layout, err := optionalStringField(generator, index, object, "layout")
	if err != nil {
		return Page{}, err
	}
	title, err := optionalStringField(generator, index, object, "title")
	if err != nil {
		return Page{}, err
	}
	content, err := optionalStringField(generator, index, object, "content")
	if err != nil {
		return Page{}, err
	}
	date, err := optionalStringField(generator, index, object, "date")
	if err != nil {
		return Page{}, err
	}
	data, err := optionalDataField(generator, index, object)
	if err != nil {
		return Page{}, err
	}

	return Page{
		Content:    content,
		Data:       data,
		Date:       date,
		Generator:  generator,
		Index:      index,
		Layout:     layout,
		OutputPath: outputPath,
		Permalink:  normalizedPermalink,
		Title:      title,
	}, nil
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

// rejectUnknownFields rejects top-level page fields outside Veta's contract.
func rejectUnknownFields(generator string, index int, object map[string]any) error {
	for field := range object {
		if _, allowed := allowedPageFields[field]; !allowed {
			return pageError(generator, index, "unknown field %q", field)
		}
	}

	return nil
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

// optionalStringField returns an optional page string field.
func optionalStringField(
	generator string,
	index int,
	object map[string]any,
	field string,
) (string, error) {
	value, ok := object[field]
	if !ok {
		return "", nil
	}

	stringValue, ok := value.(string)
	if !ok {
		return "", pageError(generator, index, "%s must be a string", field)
	}

	return stringValue, nil
}

// optionalDataField returns an optional JSON-compatible page data object.
func optionalDataField(generator string, index int, object map[string]any) (map[string]any, error) {
	value, ok := object["data"]
	if !ok {
		return map[string]any{}, nil
	}

	normalizedValue, err := normalizeJSONValue(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s[%d].data: %w", ErrPageInvalid, generator, index, err)
	}

	data, ok := normalizedValue.(map[string]any)
	if !ok || data == nil {
		return nil, pageError(generator, index, "data must be an object")
	}

	return data, nil
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
