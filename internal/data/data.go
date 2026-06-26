package data

import (
	"errors"
	"fmt"
	"io/fs"

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

	entries, err := fs.ReadDir(files, DirName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Values{}, nil
		}

		return nil, fmt.Errorf("read data directory %s: %w", DirName, err)
	}

	runner := js.New(config.jsOptions...)
	values := Values{}
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("%w: %s/%s", ErrNestedUnsupported, DirName, entry.Name())
		}

		key, extension, err := dataFileKey(entry.Name())
		if err != nil {
			return nil, err
		}

		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("%w: %s", ErrKeyDuplicate, key)
		}

		value, err := loadDataFile(files, runner, entry.Name(), extension)
		if err != nil {
			return nil, err
		}

		values[key] = value
	}

	return values, nil
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
