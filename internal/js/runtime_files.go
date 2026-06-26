package js

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/dop251/goja"
)

var (
	// ErrEmptyPath indicates that a Veta file API received an empty path.
	ErrEmptyPath = errors.New("path cannot be empty")

	// ErrPathOutsideRoot indicates that a Veta file API tried to escape the root.
	ErrPathOutsideRoot = errors.New("path must stay inside the configured root")
)

// newFileAPI returns the synchronous file APIs exposed through Veta.files.
func (r *Runner) newFileAPI(vm *goja.Runtime) (*goja.Object, error) {
	root, err := r.rootDir()
	if err != nil {
		return nil, err
	}

	api := &fileAPI{root: root, vm: vm}
	files := vm.NewObject()
	fileMethods := Runtime{
		"listFiles": api.listFiles,
		"readFile":  api.readFile,
		"readFiles": api.readFiles,
	}
	for name, value := range fileMethods {
		if err := files.Set(name, value); err != nil {
			return nil, fmt.Errorf("set %s.files.%s: %w", GlobalName, name, err)
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
	pattern, err := requiredStringArgument(call.Argument(0), "Veta.files.listFiles pattern")
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
	rawPath, err := requiredStringArgument(call.Argument(0), "Veta.files.readFile path")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	filePath, err := cleanRelativeFilePath(rawPath)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	root, err := api.openRoot()
	if err != nil {
		panic(api.vm.NewGoError(err))
	}
	defer func() {
		_ = root.Close()
	}()

	content, err := root.ReadFile(filepath.FromSlash(filePath))
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("read file %s: %w", filePath, err)))
	}

	return api.vm.ToValue(string(content))
}

// readFiles returns sorted file paths and contents matching a glob pattern.
func (api *fileAPI) readFiles(call goja.FunctionCall) goja.Value {
	pattern, err := requiredStringArgument(call.Argument(0), "Veta.files.readFiles pattern")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	matches, err := api.matchFiles(pattern)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	root, err := api.openRoot()
	if err != nil {
		panic(api.vm.NewGoError(err))
	}
	defer func() {
		_ = root.Close()
	}()

	files := make([]map[string]string, 0, len(matches))
	for _, match := range matches {
		content, err := root.ReadFile(filepath.FromSlash(match))
		if err != nil {
			panic(api.vm.NewGoError(fmt.Errorf("read file %s: %w", match, err)))
		}

		files = append(files, map[string]string{
			"content": string(content),
			"path":    match,
		})
	}

	return api.vm.ToValue(files)
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

	for _, segment := range strings.Split(rawPath, "/") {
		if segment == ".." {
			return "", ErrPathOutsideRoot
		}
	}

	cleanPath := path.Clean(rawPath)
	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", ErrPathOutsideRoot
	}

	return cleanPath, nil
}
