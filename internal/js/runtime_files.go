package js

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/dop251/goja"
	"github.com/varavelio/veta/internal/permalink"
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
		"listFiles":   api.listFiles,
		"readFile":    api.readFile,
		"toPermalink": api.toPermalink,
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
	for name := range object {
		if name != "stripPrefix" {
			return permalink.PathOptions{}, fmt.Errorf(
				"files.toPermalink unknown option %q",
				name,
			)
		}
	}
	if stripPrefix, exists := object["stripPrefix"]; exists {
		text, ok := stripPrefix.(string)
		if !ok {
			return permalink.PathOptions{}, fmt.Errorf(
				"files.toPermalink stripPrefix must be a string",
			)
		}
		options.StripPrefix = text
	}

	return options, nil
}
