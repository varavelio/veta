package data

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/dop251/goja"
	"github.com/varavelio/veta/internal/js"
	"gopkg.in/yaml.v3"
)

// loadDataFile reads, parses, and normalizes one data file.
func loadDataFile(files fs.FS, runner *js.Runner, fileName, extension string) (any, error) {
	filePath := path.Join(DirName, fileName)
	content, err := fs.ReadFile(files, filePath)
	if err != nil {
		return nil, fmt.Errorf("read data file %s: %w", filePath, err)
	}

	value, err := parseDataFile(runner, filePath, content, extension)
	if err != nil {
		return nil, fmt.Errorf("parse data file %s: %w", filePath, err)
	}

	value, err = normalizeJSONValue(value)
	if err != nil {
		return nil, fmt.Errorf("normalize data file %s: %w", filePath, err)
	}

	return value, nil
}

// parseDataFile dispatches file content to the parser selected by extension.
func parseDataFile(
	runner *js.Runner,
	filePath string,
	content []byte,
	extension string,
) (any, error) {
	switch extension {
	case ".json":
		return parseJSON(content)
	case ".yaml", ".yml":
		return parseYAML(content)
	case ".toml":
		return parseTOML(content)
	case ".js":
		return executeJavaScript(runner, filePath, content)
	default:
		return nil, fmt.Errorf("%w: %s", ErrFormatUnsupported, extension)
	}
}

// parseJSON decodes one JSON value and rejects trailing values.
func parseJSON(content []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
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

	return value, nil
}

// parseYAML decodes one YAML document into a generic value.
func parseYAML(content []byte) (any, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))

	var value any
	if err := decoder.Decode(&value); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
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

	return value, nil
}

// parseTOML decodes TOML into its top-level table value.
func parseTOML(content []byte) (any, error) {
	value := map[string]any{}
	if _, err := toml.Decode(string(content), &value); err != nil {
		return nil, fmt.Errorf("%w: decode toml: %w", ErrInvalid, err)
	}

	return value, nil
}

// executeJavaScript runs a synchronous data script and returns its exported
// value.
func executeJavaScript(runner *js.Runner, filePath string, content []byte) (any, error) {
	result, err := runner.Execute(js.Source{Name: filePath, Code: string(content)})
	if err != nil {
		return nil, fmt.Errorf("%w: execute javascript: %w", ErrInvalid, err)
	}

	if goja.IsUndefined(result.Value()) {
		return nil, fmt.Errorf("%w: javascript data must return a value", ErrValueUnsupported)
	}

	return result.Export(), nil
}
