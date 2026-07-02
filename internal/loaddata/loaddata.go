// Package loaddata loads local and remote template data sources.
package loaddata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"mime"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

const defaultHTTPTimeout = 30 * time.Second

var (
	// ErrConfigInvalid indicates that the data loader was configured incorrectly.
	ErrConfigInvalid = errors.New("load_data config is invalid")

	// ErrRequestInvalid indicates that a load_data request is malformed.
	ErrRequestInvalid = errors.New("load_data request is invalid")

	// ErrPathInvalid indicates that a local path is unsafe or malformed.
	ErrPathInvalid = errors.New("load_data path is invalid")

	// ErrURLInvalid indicates that a remote URL is unsafe or malformed.
	ErrURLInvalid = errors.New("load_data url is invalid")

	// ErrFormatInvalid indicates that a format is not supported.
	ErrFormatInvalid = errors.New("load_data format is invalid")

	// ErrHTTPStatus indicates that a remote response was not successful.
	ErrHTTPStatus = errors.New("load_data http status is not ok")
)

// Request describes one local or remote data source.
type Request struct {
	Path    string
	URL     string
	Format  string
	Timeout time.Duration
}

// Loader loads template data from local files and remote URLs.
type Loader struct {
	files       fs.FS
	httpClient  *http.Client
	httpTimeout time.Duration
}

// Option configures a Loader.
type Option func(*Loader)

// WithHTTPClient configures the HTTP client used for remote requests.
func WithHTTPClient(client *http.Client) Option {
	return func(loader *Loader) {
		loader.httpClient = client
	}
}

// WithHTTPTimeout configures the default timeout used for remote requests.
func WithHTTPTimeout(timeout time.Duration) Option {
	return func(loader *Loader) {
		loader.httpTimeout = timeout
	}
}

// New creates a Loader backed by files.
func New(files fs.FS, options ...Option) (*Loader, error) {
	if files == nil {
		return nil, fmt.Errorf("%w: filesystem is required", ErrConfigInvalid)
	}

	loader := &Loader{files: files, httpTimeout: defaultHTTPTimeout}
	for _, option := range options {
		if option != nil {
			option(loader)
		}
	}
	if loader.httpClient == nil {
		loader.httpClient = http.DefaultClient
	}
	if loader.httpTimeout <= 0 {
		loader.httpTimeout = defaultHTTPTimeout
	}

	return loader, nil
}

// Load reads and parses one local or remote data source.
func (loader *Loader) Load(request Request) (any, error) {
	if loader == nil || loader.files == nil {
		return nil, fmt.Errorf("%w: loader is nil", ErrConfigInvalid)
	}

	if strings.TrimSpace(request.Path) != "" && strings.TrimSpace(request.URL) != "" {
		return nil, fmt.Errorf("%w: use either path or url", ErrRequestInvalid)
	}
	if strings.TrimSpace(request.Path) == "" && strings.TrimSpace(request.URL) == "" {
		return nil, fmt.Errorf("%w: path or url is required", ErrRequestInvalid)
	}
	if request.Timeout < 0 {
		return nil, fmt.Errorf("%w: timeout cannot be negative", ErrRequestInvalid)
	}

	if strings.TrimSpace(request.Path) != "" {
		return loader.loadPath(request)
	}

	return loader.loadURL(request)
}

// Function returns a positional helper suitable for template function calls.
func (loader *Loader) Function() func(string, ...any) (any, error) {
	return func(source string, arguments ...any) (any, error) {
		if len(arguments) > 2 {
			return nil, fmt.Errorf(
				"%w: load_data accepts source, optional format, and optional timeout_ms",
				ErrRequestInvalid,
			)
		}

		request := Request{Path: source}
		if len(arguments) >= 1 {
			format, ok := arguments[0].(string)
			if !ok {
				return nil, fmt.Errorf("%w: format must be a string", ErrRequestInvalid)
			}
			request.Format = format
		}
		if len(arguments) == 2 {
			timeout, err := durationFromMilliseconds(arguments[1])
			if err != nil {
				return nil, err
			}
			request.Timeout = timeout
		}

		if isRemoteURL(source) {
			request.Path = ""
			request.URL = source
		}

		return loader.Load(request)
	}
}

func durationFromMilliseconds(value any) (time.Duration, error) {
	integer := reflect.ValueOf(value)
	if !integer.IsValid() {
		return 0, fmt.Errorf("%w: timeout_ms must be an integer", ErrRequestInvalid)
	}

	switch integer.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		milliseconds := integer.Int()
		if milliseconds < 0 {
			return 0, fmt.Errorf("%w: timeout_ms cannot be negative", ErrRequestInvalid)
		}
		return time.Duration(milliseconds) * time.Millisecond, nil
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		milliseconds := integer.Uint()
		if milliseconds > uint64(math.MaxInt64/int64(time.Millisecond)) {
			return 0, fmt.Errorf("%w: timeout_ms is too large", ErrRequestInvalid)
		}
		return time.Duration(milliseconds) * time.Millisecond, nil
	default:
		return 0, fmt.Errorf("%w: timeout_ms must be an integer", ErrRequestInvalid)
	}
}

func (loader *Loader) loadPath(request Request) (any, error) {
	filePath, err := cleanRelativePath(request.Path)
	if err != nil {
		return nil, err
	}

	content, err := fs.ReadFile(loader.files, filePath)
	if err != nil {
		return nil, fmt.Errorf("read load_data path %s: %w", filePath, err)
	}

	format, err := dataFormat(request.Format, filePath, "")
	if err != nil {
		return nil, err
	}

	value, err := parseContent(content, format)
	if err != nil {
		return nil, fmt.Errorf("parse load_data path %s: %w", filePath, err)
	}

	return value, nil
}

func (loader *Loader) loadURL(request Request) (any, error) {
	cleanURL, err := cleanURL(request.URL)
	if err != nil {
		return nil, err
	}

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = loader.httpTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, cleanURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create load_data request: %w", err)
	}
	response, err := loader.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("execute load_data request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read load_data response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("%w: %s returned %d", ErrHTTPStatus, cleanURL, response.StatusCode)
	}

	format, err := dataFormat(
		request.Format,
		response.Request.URL.Path,
		response.Header.Get("Content-Type"),
	)
	if err != nil {
		return nil, err
	}

	value, err := parseContent(content, format)
	if err != nil {
		return nil, fmt.Errorf("parse load_data url %s: %w", cleanURL, err)
	}

	return value, nil
}

func cleanRelativePath(rawPath string) (string, error) {
	name := strings.TrimSpace(rawPath)
	if name == "" || strings.ContainsRune(name, 0) || filepath.VolumeName(name) != "" ||
		hasWindowsVolumeName(name) || filepath.IsAbs(name) {
		return "", ErrPathInvalid
	}

	name = strings.ReplaceAll(name, "\\", "/")
	if path.IsAbs(name) || slices.Contains(strings.Split(name, "/"), "..") {
		return "", ErrPathInvalid
	}

	cleanPath := path.Clean(name)
	if cleanPath == "." || !fs.ValidPath(cleanPath) {
		return "", ErrPathInvalid
	}

	return cleanPath, nil
}

func cleanURL(rawURL string) (string, error) {
	text := strings.TrimSpace(rawURL)
	if text == "" || strings.ContainsRune(text, 0) {
		return "", ErrURLInvalid
	}

	parsedURL, err := url.Parse(text)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrURLInvalid, err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return "", ErrURLInvalid
	}

	return parsedURL.String(), nil
}

func isRemoteURL(source string) bool {
	parsedURL, err := url.Parse(strings.TrimSpace(source))
	if err != nil {
		return false
	}

	return (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") && parsedURL.Host != ""
}

func dataFormat(rawFormat, sourcePath, contentType string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(rawFormat))
	if format == "" {
		format = formatFromContentType(contentType)
	}
	if format == "" {
		format = formatFromPath(sourcePath)
	}
	if format == "" {
		format = "text"
	}

	switch format {
	case "text", "json", "yaml", "toml":
		return format, nil
	case "yml":
		return "yaml", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrFormatInvalid, rawFormat)
	}
}

func formatFromPath(sourcePath string) string {
	switch strings.ToLower(path.Ext(sourcePath)) {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".txt", ".text":
		return "text"
	default:
		return "text"
	}
}

func formatFromContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	mediaType = strings.ToLower(mediaType)

	switch {
	case mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"):
		return "json"
	case mediaType == "application/yaml" || mediaType == "application/x-yaml" ||
		mediaType == "text/yaml" || mediaType == "text/x-yaml":
		return "yaml"
	case mediaType == "application/toml" || mediaType == "application/toml+text":
		return "toml"
	case strings.HasPrefix(mediaType, "text/"):
		return "text"
	default:
		return ""
	}
}

func parseContent(content []byte, format string) (any, error) {
	switch format {
	case "text":
		return string(content), nil
	case "json":
		return parseJSONValue(content)
	case "yaml":
		return parseYAMLValue(content)
	case "toml":
		return parseTOMLValue(content)
	default:
		return nil, fmt.Errorf("%w: %s", ErrFormatInvalid, format)
	}
}

func parseJSONValue(content []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("decode json: multiple json documents are not supported")
		}

		return nil, fmt.Errorf("decode json: %w", err)
	}

	return normalizeStructuredValue(value)
}

func parseYAMLValue(content []byte) (any, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("decode yaml: multiple yaml documents are not supported")
		}

		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	return normalizeStructuredValue(value)
}

func parseTOMLValue(content []byte) (any, error) {
	var value any
	if err := toml.Unmarshal(content, &value); err != nil {
		return nil, fmt.Errorf("decode toml: %w", err)
	}

	return normalizeStructuredValue(value)
}

func normalizeStructuredValue(value any) (any, error) {
	return normalizeStructuredValueAt(value, "$.")
}

func normalizeStructuredValueAt(value any, location string) (any, error) {
	switch typedValue := value.(type) {
	case nil:
		return nil, nil
	case string, bool:
		return typedValue, nil
	case []any:
		items := make([]any, len(typedValue))
		for index, item := range typedValue {
			normalized, err := normalizeStructuredValueAt(
				item,
				fmt.Sprintf("%s[%d]", location, index),
			)
			if err != nil {
				return nil, err
			}
			items[index] = normalized
		}

		return items, nil
	case map[string]any:
		items := make(map[string]any, len(typedValue))
		for key, item := range typedValue {
			normalized, err := normalizeStructuredValueAt(item, location+"."+key)
			if err != nil {
				return nil, err
			}
			items[key] = normalized
		}

		return items, nil
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

func normalizeStructuredFloat(value float64, location string) (float64, error) {
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, fmt.Errorf("%s has non-finite number", location)
	}

	return value, nil
}

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

func hasWindowsVolumeName(name string) bool {
	return len(name) >= 2 && name[1] == ':' &&
		('A' <= name[0] && name[0] <= 'Z' || 'a' <= name[0] && name[0] <= 'z')
}
