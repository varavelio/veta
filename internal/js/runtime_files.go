package js

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/dop251/goja"
	"github.com/varavelio/veta/internal/permalink"
	"gopkg.in/yaml.v3"
)

var (
	// ErrEmptyPath indicates that a Veta file API received an empty path.
	ErrEmptyPath = errors.New("path cannot be empty")

	// ErrPathOutsideRoot indicates that a Veta file API tried to escape the root.
	ErrPathOutsideRoot = errors.New("path must stay inside the configured root")
)

// newFileAPI returns the synchronous file APIs exposed through context.files.
func (r *Runner) newFileAPI(vm *goja.Runtime) (*goja.Object, error) {
	root, err := r.rootDir()
	if err != nil {
		return nil, err
	}

	api := &fileAPI{root: root, vm: vm}
	files := vm.NewObject()
	fileMethods := Runtime{
		"listFiles":        api.listFiles,
		"readFile":         api.readFile,
		"readJsonFile":     api.readJSONFile,
		"readMarkdownFile": api.readMarkdownFile,
		"readTomlFile":     api.readTOMLFile,
		"readYamlFile":     api.readYAMLFile,
		"toPermalink":      api.toPermalink,
	}
	for name, value := range fileMethods {
		if err := files.Set(name, value); err != nil {
			return nil, fmt.Errorf("set %s.files.%s: %w", runtimeObjectName, name, err)
		}
	}

	return files, nil
}

// rootDir returns the absolute root used by Veta file APIs.
func (r *Runner) rootDir() (string, error) {
	root := defaultRootDir
	if r != nil && strings.TrimSpace(r.root) != "" {
		root = r.root
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root directory: %w", err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("stat root directory %s: %w", absRoot, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("root is not a directory: %s", absRoot)
	}

	return absRoot, nil
}

// fileAPI owns synchronous file callbacks exposed to JavaScript.
type fileAPI struct {
	root string
	vm   *goja.Runtime
}

// listFiles returns sorted files matching a glob pattern inside the root.
func (api *fileAPI) listFiles(call goja.FunctionCall) goja.Value {
	pattern, err := requiredStringArgument(call.Argument(0), "files.listFiles pattern")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	matches, err := api.matchFiles(pattern)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	return api.vm.ToValue(matches)
}

// readFile returns one file as a UTF-8 string.
func (api *fileAPI) readFile(call goja.FunctionCall) goja.Value {
	rawPath, err := requiredStringArgument(call.Argument(0), "files.readFile path")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	_, content, err := api.readProjectFile(rawPath)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	return api.vm.ToValue(string(content))
}

// readMarkdownFile returns one Markdown file with parsed front matter.
func (api *fileAPI) readMarkdownFile(call goja.FunctionCall) goja.Value {
	rawPath, err := requiredStringArgument(call.Argument(0), "files.readMarkdownFile path")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	filePath, content, err := api.readProjectFile(rawPath)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	document, err := parseMarkdownDocument(string(content))
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("parse markdown file %s: %w", filePath, err)))
	}
	document["path"] = filePath

	return api.vm.ToValue(document)
}

// readJSONFile returns one parsed JSON file.
func (api *fileAPI) readJSONFile(call goja.FunctionCall) goja.Value {
	rawPath, err := requiredStringArgument(call.Argument(0), "files.readJsonFile path")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	filePath, content, err := api.readProjectFile(rawPath)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	value, err := parseJSONValue(content)
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("parse json file %s: %w", filePath, err)))
	}

	return api.vm.ToValue(value)
}

// readYAMLFile returns one parsed YAML file.
func (api *fileAPI) readYAMLFile(call goja.FunctionCall) goja.Value {
	rawPath, err := requiredStringArgument(call.Argument(0), "files.readYamlFile path")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	filePath, content, err := api.readProjectFile(rawPath)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	value, err := parseYAMLValue(content)
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("parse yaml file %s: %w", filePath, err)))
	}

	return api.vm.ToValue(value)
}

// readTOMLFile returns one parsed TOML file.
func (api *fileAPI) readTOMLFile(call goja.FunctionCall) goja.Value {
	rawPath, err := requiredStringArgument(call.Argument(0), "files.readTomlFile path")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	filePath, content, err := api.readProjectFile(rawPath)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	value, err := parseTOMLValue(content)
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("parse toml file %s: %w", filePath, err)))
	}

	return api.vm.ToValue(value)
}

// toPermalink converts a project-relative file path into a pretty permalink.
func (api *fileAPI) toPermalink(call goja.FunctionCall) goja.Value {
	rawPath, err := requiredStringArgument(call.Argument(0), "files.toPermalink path")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	options, err := permalinkPathOptions(call.Argument(1))
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	value, err := permalink.FromPath(rawPath, options)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	return api.vm.ToValue(value)
}

// openRoot opens the configured root with the standard-library confinement
// guard, including symlink escape protection.
func (api *fileAPI) openRoot() (*os.Root, error) {
	root, err := os.OpenRoot(api.root)
	if err != nil {
		return nil, fmt.Errorf("open root directory %s: %w", api.root, err)
	}

	return root, nil
}

// readProjectFile reads one safely-normalized project-relative file.
func (api *fileAPI) readProjectFile(rawPath string) (string, []byte, error) {
	filePath, err := cleanRelativeFilePath(rawPath)
	if err != nil {
		return "", nil, err
	}

	root, err := api.openRoot()
	if err != nil {
		return "", nil, err
	}
	defer func() {
		_ = root.Close()
	}()

	content, err := root.ReadFile(filepath.FromSlash(filePath))
	if err != nil {
		return "", nil, fmt.Errorf("read file %s: %w", filePath, err)
	}

	return filePath, content, nil
}

// matchFiles returns sorted file matches for a sanitized glob pattern.
func (api *fileAPI) matchFiles(pattern string) ([]string, error) {
	cleanPattern, err := cleanRelativeGlobPattern(pattern)
	if err != nil {
		return nil, err
	}

	root, err := api.openRoot()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = root.Close()
	}()

	matches, err := doublestar.Glob(
		root.FS(),
		cleanPattern,
		doublestar.WithFilesOnly(),
		doublestar.WithFailOnIOErrors(),
	)
	if err != nil {
		return nil, fmt.Errorf("list files matching %s: %w", cleanPattern, err)
	}

	sort.Strings(matches)

	return matches, nil
}

// cleanRelativeFilePath normalizes a user-provided file path.
func cleanRelativeFilePath(filePath string) (string, error) {
	cleanPath, err := cleanRelativePath(filePath)
	if err != nil {
		return "", err
	}

	if cleanPath == "." {
		return "", ErrEmptyPath
	}

	return cleanPath, nil
}

// cleanRelativeGlobPattern normalizes a user-provided glob pattern.
func cleanRelativeGlobPattern(pattern string) (string, error) {
	cleanPattern, err := cleanRelativePath(pattern)
	if err != nil {
		return "", err
	}

	if cleanPattern == "." {
		return "**/*", nil
	}

	return cleanPattern, nil
}

// cleanRelativePath normalizes a slash-separated path while rejecting escapes.
func cleanRelativePath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(filepath.ToSlash(rawPath))
	if rawPath == "" {
		return "", ErrEmptyPath
	}

	if path.IsAbs(rawPath) || filepath.IsAbs(rawPath) || filepath.VolumeName(rawPath) != "" {
		return "", ErrPathOutsideRoot
	}

	for strings.HasPrefix(rawPath, "./") {
		rawPath = strings.TrimPrefix(rawPath, "./")
	}

	if slices.Contains(strings.Split(rawPath, "/"), "..") {
		return "", ErrPathOutsideRoot
	}

	cleanPath := path.Clean(rawPath)
	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", ErrPathOutsideRoot
	}

	return cleanPath, nil
}

// permalinkPathOptions converts an optional JavaScript object into path options.
func permalinkPathOptions(value goja.Value) (permalink.PathOptions, error) {
	if isJavaScriptNullish(value) {
		return permalink.PathOptions{}, nil
	}

	object, ok := value.Export().(map[string]any)
	if !ok {
		return permalink.PathOptions{}, fmt.Errorf("files.toPermalink options must be an object")
	}

	options := permalink.PathOptions{}
	if basePath, exists := object["basePath"]; exists {
		text, ok := basePath.(string)
		if !ok {
			return permalink.PathOptions{}, fmt.Errorf(
				"files.toPermalink basePath must be a string",
			)
		}
		options.BasePath = text
	}

	return options, nil
}

// parseMarkdownDocument splits optional YAML or TOML front matter from Markdown.
func parseMarkdownDocument(content string) (map[string]any, error) {
	delimiter, frontMatter, body, found, err := splitFrontMatter(content)
	if err != nil {
		return nil, err
	}
	if !found {
		return map[string]any{"content": content, "frontmatter": map[string]any{}}, nil
	}

	var value any
	switch delimiter {
	case "---":
		value, err = parseYAMLValue([]byte(frontMatter))
	case "+++":
		value, err = parseTOMLValue([]byte(frontMatter))
	default:
		err = fmt.Errorf("unsupported front matter delimiter %q", delimiter)
	}
	if err != nil {
		return nil, err
	}

	frontMatterObject, ok := value.(map[string]any)
	if !ok || frontMatterObject == nil {
		return nil, fmt.Errorf("front matter must be an object")
	}

	return map[string]any{
		"content":     trimLeadingBlankLine(body),
		"frontmatter": frontMatterObject,
	}, nil
}

// splitFrontMatter extracts a leading front matter block when one exists.
func splitFrontMatter(content string) (string, string, string, bool, error) {
	firstLine, rest := nextLine(content)
	if firstLine != "---" && firstLine != "+++" {
		return "", "", content, false, nil
	}

	delimiter := firstLine
	frontMatterStart := len(content) - len(rest)
	remaining := rest
	for len(remaining) > 0 {
		line, next := nextLine(remaining)
		lineStart := len(content) - len(remaining)
		if line == delimiter {
			return delimiter, content[frontMatterStart:lineStart], next, true, nil
		}

		remaining = next
	}

	return "", "", "", false, fmt.Errorf("unterminated front matter block")
}

// nextLine returns one line without its newline and the remaining content.
func nextLine(content string) (string, string) {
	line, rest, ok := strings.Cut(content, "\n")
	if !ok {
		return strings.TrimSuffix(content, "\r"), ""
	}

	return strings.TrimSuffix(line, "\r"), rest
}

// trimLeadingBlankLine removes one blank separator after front matter.
func trimLeadingBlankLine(content string) string {
	if trimmed, ok := strings.CutPrefix(content, "\r\n"); ok {
		return trimmed
	}

	trimmed, _ := strings.CutPrefix(content, "\n")
	return trimmed
}

// parseJSONValue decodes one JSON value.
func parseJSONValue(content []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("multiple json values are not supported")
		}

		return nil, fmt.Errorf("decode json: %w", err)
	}

	return normalizeStructuredValue(value)
}

// parseYAMLValue decodes one YAML document.
func parseYAMLValue(content []byte) (any, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))

	var value any
	if err := decoder.Decode(&value); err != nil {
		if errors.Is(err, io.EOF) {
			return map[string]any{}, nil
		}

		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("multiple yaml documents are not supported")
		}

		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	return normalizeStructuredValue(value)
}

// parseTOMLValue decodes one TOML document.
func parseTOMLValue(content []byte) (any, error) {
	value := map[string]any{}
	if _, err := toml.Decode(string(content), &value); err != nil {
		return nil, fmt.Errorf("decode toml: %w", err)
	}

	return normalizeStructuredValue(value)
}

// normalizeStructuredValue converts parsed data into JSON-compatible values.
func normalizeStructuredValue(value any) (any, error) {
	return normalizeStructuredValueAt(value, "$")
}

// normalizeStructuredValueAt converts parsed data while tracking error location.
func normalizeStructuredValueAt(value any, location string) (any, error) {
	switch typedValue := value.(type) {
	case nil:
		return nil, nil
	case bool, string:
		return typedValue, nil
	case json.Number:
		return normalizeStructuredNumber(typedValue, location)
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
		return normalizeStructuredFloat(float64(typedValue), location)
	case float64:
		return normalizeStructuredFloat(typedValue, location)
	case time.Time:
		return typedValue.Format(time.RFC3339Nano), nil
	}

	return normalizeReflectedStructuredValue(reflect.ValueOf(value), location)
}

// normalizeStructuredNumber converts a JSON number to an integer when possible.
func normalizeStructuredNumber(number json.Number, location string) (any, error) {
	if integer, err := number.Int64(); err == nil {
		return integer, nil
	}

	float, err := number.Float64()
	if err != nil {
		return nil, fmt.Errorf("%s has invalid number %q", location, number)
	}

	return normalizeStructuredFloat(float, location)
}

// normalizeStructuredFloat rejects non-finite numbers.
func normalizeStructuredFloat(value float64, location string) (float64, error) {
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, fmt.Errorf("%s has non-finite number", location)
	}

	return value, nil
}

// normalizeReflectedStructuredValue handles maps and slices from parsers.
func normalizeReflectedStructuredValue(value reflect.Value, location string) (any, error) {
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
		return normalizeStructuredMap(value, location)
	case reflect.Slice, reflect.Array:
		return normalizeStructuredSlice(value, location)
	default:
		return nil, fmt.Errorf("%s has unsupported value type %s", location, value.Type())
	}
}

// normalizeStructuredMap converts a parsed map into map[string]any.
func normalizeStructuredMap(value reflect.Value, location string) (map[string]any, error) {
	if value.IsNil() {
		return nil, nil
	}

	items := make(map[string]any, value.Len())
	iterator := value.MapRange()
	for iterator.Next() {
		key, ok := structuredStringKey(iterator.Key())
		if !ok {
			return nil, fmt.Errorf("%s has non-string map key", location)
		}

		item, err := normalizeStructuredValueAt(iterator.Value().Interface(), location+"."+key)
		if err != nil {
			return nil, err
		}

		items[key] = item
	}

	return items, nil
}

// structuredStringKey extracts a string from a reflected map key.
func structuredStringKey(value reflect.Value) (string, bool) {
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

// normalizeStructuredSlice converts a parsed slice into []any.
func normalizeStructuredSlice(value reflect.Value, location string) ([]any, error) {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return nil, nil
	}

	items := make([]any, value.Len())
	for index := range value.Len() {
		item, err := normalizeStructuredValueAt(
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
