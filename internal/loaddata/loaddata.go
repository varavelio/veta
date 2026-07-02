// Package loaddata loads local and remote template data sources.
package loaddata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
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

	// ErrHTTPStatus indicates that a remote response was not successful.
	ErrHTTPStatus = errors.New("load_data http status is not ok")
)

// Request describes one local or remote data source.
type Request struct {
	Path    string
	URL     string
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
func (loader *Loader) Function() func(string) (any, error) {
	return func(source string) (any, error) {
		request := Request{Path: source}
		if isRemoteURL(source) {
			request.Path = ""
			request.URL = source
		}

		return loader.Load(request)
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

	return string(content), nil
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

	return string(content), nil
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

func hasWindowsVolumeName(name string) bool {
	return len(name) >= 2 && name[1] == ':' &&
		('A' <= name[0] && name[0] <= 'Z' || 'a' <= name[0] && name[0] <= 'z')
}
