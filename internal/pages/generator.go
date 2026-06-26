package pages

import (
	"fmt"
	"io/fs"
	"path"
	"reflect"

	"github.com/dop251/goja"
	"github.com/varavelio/veta/internal/js"
)

// loadGenerator executes one generator file and decodes its returned pages.
func loadGenerator(files fs.FS, runner *js.Runner, fileName string) ([]Page, error) {
	filePath := path.Join(DirName, fileName)
	content, err := fs.ReadFile(files, filePath)
	if err != nil {
		return nil, fmt.Errorf("read page generator %s: %w", filePath, err)
	}

	result, err := runner.Execute(js.Source{Name: filePath, Code: string(content)})
	if err != nil {
		return nil, fmt.Errorf("%w: execute %s: %w", ErrGeneratorInvalid, filePath, err)
	}

	if goja.IsUndefined(result.Value()) {
		return nil, fmt.Errorf("%w: %s must return an array", ErrGeneratorInvalid, filePath)
	}

	return decodeGeneratorPages(fileName, result.Export())
}

// decodeGeneratorPages converts one generator result into normalized pages.
func decodeGeneratorPages(generator string, value any) ([]Page, error) {
	items, ok := sliceItems(value)
	if !ok {
		return nil, fmt.Errorf("%w: %s must return an array", ErrGeneratorInvalid, generator)
	}

	pages := make([]Page, 0, len(items))
	for index, item := range items {
		page, err := decodePage(generator, index, item)
		if err != nil {
			return nil, err
		}

		pages = append(pages, page)
	}

	return pages, nil
}

// sliceItems converts a reflected slice or array into []any.
func sliceItems(value any) ([]any, bool) {
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

	switch reflectedValue.Kind() {
	case reflect.Slice, reflect.Array:
		items := make([]any, reflectedValue.Len())
		for index := range reflectedValue.Len() {
			items[index] = reflectedValue.Index(index).Interface()
		}

		return items, true
	default:
		return nil, false
	}
}
