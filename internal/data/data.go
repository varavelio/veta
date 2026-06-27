package data

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/varavelio/veta/internal/js"
)

// DirName is the project directory containing global data files.
const DirName = "data"

// Values contains global data keyed by file name without extension.
type Values map[string]any

// Option configures data loading.
type Option func(*loadConfig) error

type loadConfig struct {
	jsOptions []js.Option
}

// WithJSOptions configures the JavaScript runner used for data files ending in
// .js.
func WithJSOptions(options ...js.Option) Option {
	return func(config *loadConfig) error {
		for _, option := range options {
			if option == nil {
				continue
			}

			config.jsOptions = append(config.jsOptions, option)
		}

		return nil
	}
}

// Load reads global data files from the data directory. Missing data directories
// return an empty Values map.
func Load(files fs.FS, options ...Option) (Values, error) {
	if files == nil {
		return nil, ErrFSRequired
	}

	config, err := newLoadConfig(options)
	if err != nil {
		return nil, err
	}

	runner := js.New(config.jsOptions...)
	values := Values{}
	if err := fs.WalkDir(files, DirName, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == DirName || entry.IsDir() {
			return nil
		}

		relativeName := strings.TrimPrefix(name, DirName+"/")
		keys, extension, err := dataFileKey(relativeName)
		if err != nil {
			return err
		}

		value, err := loadDataFile(files, runner, relativeName, extension)
		if err != nil {
			return err
		}

		return setNestedValue(values, keys, value)
	}); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Values{}, nil
		}

		return nil, fmt.Errorf("walk data directory %s: %w", DirName, err)
	}

	return values, nil
}

// setNestedValue inserts a parsed data value into a nested data namespace.
func setNestedValue(values Values, keys []string, value any) error {
	current := map[string]any(values)
	for _, key := range keys[:len(keys)-1] {
		next, exists := current[key]
		if !exists {
			nested := map[string]any{}
			current[key] = nested
			current = nested
			continue
		}

		nested, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: %s", ErrKeyDuplicate, path.Join(keys...))
		}
		current = nested
	}

	key := keys[len(keys)-1]
	if _, exists := current[key]; exists {
		return fmt.Errorf("%w: %s", ErrKeyDuplicate, path.Join(keys...))
	}

	current[key] = value
	return nil
}

// newLoadConfig applies loader options into a configuration value.
func newLoadConfig(options []Option) (loadConfig, error) {
	var config loadConfig
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return loadConfig{}, err
		}
	}

	return config, nil
}
