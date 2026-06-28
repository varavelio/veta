package pages

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/varavelio/veta/internal/js"
)

// DirName is the project directory containing page generator scripts.
const DirName = "pages"

// Manifest contains every page discovered from page generators.
type Manifest struct {
	Pages []Page
}

// Page contains one normalized page returned by a generator.
type Page struct {
	Generator string
	Index     int

	Permalink  string
	OutputPath string

	Template string
	// Fields contains the page object exposed to templates as page.
	Fields map[string]any
}

// Option configures page loading.
type Option func(*loadConfig) error

type loadConfig struct {
	jsOptions []js.Option
}

// WithJSOptions configures the JavaScript runner used for page generators.
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

// Load executes page generators from the pages directory. Missing pages
// directories return an empty Manifest.
func Load(files fs.FS, options ...Option) (Manifest, error) {
	if files == nil {
		return Manifest{}, ErrFSRequired
	}

	config, err := newLoadConfig(options)
	if err != nil {
		return Manifest{}, err
	}

	entries, err := fs.ReadDir(files, DirName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Manifest{}, nil
		}

		return Manifest{}, fmt.Errorf("read pages directory %s: %w", DirName, err)
	}

	runner := js.New(config.jsOptions...)
	manifest := Manifest{}
	claimedOutputs := map[string]Page{}
	for _, entry := range entries {
		if entry.IsDir() {
			return Manifest{}, fmt.Errorf("%w: %s/%s", ErrNestedUnsupported, DirName, entry.Name())
		}

		if err := validateGeneratorFileName(entry.Name()); err != nil {
			return Manifest{}, err
		}

		generatorPages, err := loadGenerator(files, runner, entry.Name())
		if err != nil {
			return Manifest{}, err
		}

		for _, page := range generatorPages {
			if previousPage, exists := claimedOutputs[page.OutputPath]; exists {
				return Manifest{}, duplicateOutputError(previousPage, page)
			}

			claimedOutputs[page.OutputPath] = page
			manifest.Pages = append(manifest.Pages, page)
		}
	}

	return manifest, nil
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

// validateGeneratorFileName checks that a pages directory entry is a JavaScript
// generator file.
func validateGeneratorFileName(fileName string) error {
	if strings.ToLower(path.Ext(fileName)) != ".js" {
		return fmt.Errorf("%w: %s", ErrFormatUnsupported, path.Join(DirName, fileName))
	}

	return nil
}

// duplicateOutputError returns a diagnostic for two pages claiming one output
// path.
func duplicateOutputError(previousPage, page Page) error {
	return fmt.Errorf(
		"%w: %s from %s[%d] conflicts with %s[%d]",
		ErrOutputPathDuplicate,
		page.OutputPath,
		previousPage.Generator,
		previousPage.Index,
		page.Generator,
		page.Index,
	)
}
