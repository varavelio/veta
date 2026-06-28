package pages

import (
	"fmt"
	"io/fs"
	"maps"
	"path"
	"reflect"
	"slices"
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

	template, err := optionalTemplateField(generator, index, object)
	if err != nil {
		return Page{}, err
	}
	if _, err := requiredStringField(generator, index, object, "content"); err != nil {
		return Page{}, err
	}

	fields := clonePageFields(object)
	fields["generator"] = generator
	fields["index"] = int64(index)
	fields["outputPath"] = outputPath
	fields["permalink"] = normalizedPermalink
	fields["template"] = template

	return Page{
		Fields:     fields,
		Generator:  generator,
		Index:      index,
		OutputPath: outputPath,
		Permalink:  normalizedPermalink,
		Template:   template,
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

// optionalTemplateField returns the normalized optional page template field.
func optionalTemplateField(generator string, index int, object map[string]any) (string, error) {
	if _, exists := object["layout"]; exists {
		return "", pageError(generator, index, "layout has been renamed to template")
	}

	value, exists := object["template"]
	if !exists {
		return "", nil
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", pageError(generator, index, "template must be a string")
	}

	template := strings.TrimSpace(strings.ReplaceAll(stringValue, "\\", "/"))
	if template == "" {
		return "", pageError(
			generator,
			index,
			"template cannot be empty; omit template for raw content",
		)
	}
	if strings.ContainsRune(template, 0) || path.IsAbs(template) ||
		slices.Contains(strings.Split(template, "/"), "..") {
		return "", pageError(generator, index, "template must be relative to templates/")
	}

	template = path.Clean(template)
	if template == "." || !fs.ValidPath(template) {
		return "", pageError(generator, index, "template must be relative to templates/")
	}
	if template == "templates" || strings.HasPrefix(template, "templates/") {
		return "", pageError(
			generator,
			index,
			"template is already relative to templates/; omit the templates/ prefix",
		)
	}

	return template, nil
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
