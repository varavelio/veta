package theme

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/varavelio/veta/internal/vfs"
)

var allowedThemeDirs = []string{"templates", "components", "filters", "data", "public"}

// Site contains the project and optional theme filesystems composed for a build.
type Site struct {
	Files   fs.FS
	Project fs.FS
	Theme   fs.FS
	Source  string
}

// Option configures theme resolution.
type Option func(*resolverConfig) error

type resolverConfig struct {
	root string
}

// WithRoot configures the project root used to resolve relative theme sources.
func WithRoot(root string) Option {
	return func(config *resolverConfig) error {
		root = strings.TrimSpace(root)
		if root == "" || strings.ContainsRune(root, 0) {
			return ErrRootInvalid
		}

		config.root = root
		return nil
	}
}

// Resolve composes projectFiles with the optional local theme source.
func Resolve(projectFiles fs.FS, source string, options ...Option) (Site, error) {
	if projectFiles == nil {
		return Site{}, ErrProjectFSRequired
	}

	config, err := newResolverConfig(options)
	if err != nil {
		return Site{}, err
	}

	source = strings.TrimSpace(source)
	if source == "" {
		return Site{Files: projectFiles, Project: projectFiles}, nil
	}
	if strings.ContainsRune(source, 0) {
		return Site{}, fmt.Errorf("%w: source cannot contain NUL", ErrSourceInvalid)
	}
	if remoteSource(source) {
		return Site{}, fmt.Errorf("%w: %s", ErrRemoteUnsupported, source)
	}

	themeRoot := localThemeRoot(config.root, source)
	themeInfo, err := os.Stat(themeRoot)
	if err != nil {
		return Site{}, fmt.Errorf("%w: %s: %w", ErrSourceInvalid, source, err)
	}
	if !themeInfo.IsDir() {
		return Site{}, fmt.Errorf("%w: %s is not a directory", ErrSourceInvalid, source)
	}

	themeFiles, err := vfs.AllowTopDirs(os.DirFS(themeRoot), allowedThemeDirs...)
	if err != nil {
		return Site{}, fmt.Errorf("filter theme %s: %w", source, err)
	}
	overlay, err := vfs.NewOverlay(
		vfs.Layer{Name: "theme", FS: themeFiles},
		vfs.Layer{Name: "project", FS: projectFiles},
	)
	if err != nil {
		return Site{}, fmt.Errorf("compose theme %s: %w", source, err)
	}

	return Site{Files: overlay, Project: projectFiles, Theme: themeFiles, Source: themeRoot}, nil
}

// newResolverConfig applies options and defaults.
func newResolverConfig(options []Option) (resolverConfig, error) {
	config := resolverConfig{root: "."}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return resolverConfig{}, err
		}
	}

	return config, nil
}

// localThemeRoot returns the filesystem path for a local theme source.
func localThemeRoot(root, source string) string {
	if filepath.IsAbs(source) {
		return filepath.Clean(source)
	}

	return filepath.Clean(filepath.Join(root, source))
}

// remoteSource reports whether source looks like a remote theme reference.
func remoteSource(source string) bool {
	if strings.Contains(source, "://") {
		return true
	}
	if strings.Contains(source, "@") && !strings.HasPrefix(source, ".") &&
		!filepath.IsAbs(source) {
		return true
	}

	return false
}
