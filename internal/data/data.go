package data

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"sort"
	"strings"

	"github.com/varavelio/veta/internal/js"
	"github.com/varavelio/veta/internal/sourcefile"
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

type dataFile struct {
	extension    string
	files        fs.FS
	keys         []string
	relativeName string
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

	return LoadLayers([]fs.FS{files}, options...)
}

// LoadLayers reads global data from ordered filesystem layers. Later layers
// replace earlier files that produce the same extensionless data key.
func LoadLayers(layers []fs.FS, options ...Option) (Values, error) {
	hasLayer := false
	for _, files := range layers {
		if files != nil {
			hasLayer = true
			break
		}
	}
	if !hasLayer {
		return nil, ErrFSRequired
	}

	config, err := newLoadConfig(options)
	if err != nil {
		return nil, err
	}

	filesByKey := map[string]dataFile{}
	for _, files := range layers {
		if files == nil {
			continue
		}

		layerFiles, err := discoverDataFiles(files)
		if err != nil {
			return nil, err
		}
		maps.Copy(filesByKey, layerFiles)
	}
	if err := validateDataNamespaces(filesByKey); err != nil {
		return nil, err
	}

	logicalKeys := make([]string, 0, len(filesByKey))
	for key := range filesByKey {
		logicalKeys = append(logicalKeys, key)
	}
	sort.Strings(logicalKeys)

	runner := js.New(config.jsOptions...)
	values := Values{}
	for _, logicalKey := range logicalKeys {
		file := filesByKey[logicalKey]
		value, err := loadDataFile(file.files, runner, file.relativeName, file.extension)
		if err != nil {
			return nil, err
		}
		if err := setNestedValue(values, file.keys, value); err != nil {
			return nil, err
		}
	}

	return values, nil
}

// discoverDataFiles validates one filesystem layer and indexes its files by
// extensionless data key.
func discoverDataFiles(files fs.FS) (map[string]dataFile, error) {
	filesByKey := map[string]dataFile{}
	if err := fs.WalkDir(files, DirName, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == DirName || entry.IsDir() {
			return nil
		}
		if !sourcefile.Allowed(path.Base(name)) {
			return nil
		}

		relativeName := strings.TrimPrefix(name, DirName+"/")
		keys, extension, err := dataFileKey(relativeName)
		if err != nil {
			return err
		}

		logicalKey := path.Join(keys...)
		if _, exists := filesByKey[logicalKey]; exists {
			return fmt.Errorf("%w: %s", ErrKeyDuplicate, logicalKey)
		}

		filesByKey[logicalKey] = dataFile{
			extension:    extension,
			files:        files,
			keys:         keys,
			relativeName: relativeName,
		}
		return nil
	}); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return filesByKey, nil
		}

		return nil, fmt.Errorf("walk data directory %s: %w", DirName, err)
	}

	return filesByKey, nil
}

// validateDataNamespaces rejects a data file whose key is also used as a
// namespace by another file.
func validateDataNamespaces(filesByKey map[string]dataFile) error {
	for logicalKey := range filesByKey {
		for parent := path.Dir(logicalKey); parent != "."; parent = path.Dir(parent) {
			if _, exists := filesByKey[parent]; exists {
				return fmt.Errorf("%w: %s", ErrKeyDuplicate, parent)
			}
		}
	}

	return nil
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
