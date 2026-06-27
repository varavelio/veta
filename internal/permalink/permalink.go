package permalink

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"
)

// ErrInvalid indicates that a permalink or source path cannot be normalized.
var ErrInvalid = errors.New("permalink is invalid")

// PathOptions controls how project-relative paths become permalinks.
type PathOptions struct {
	// BasePath is stripped from the source path before generating the permalink.
	BasePath string
}

// Normalize converts a user permalink into a canonical permalink and relative
// output file path.
func Normalize(rawPermalink string) (string, string, error) {
	permalink := strings.TrimSpace(rawPermalink)
	if permalink == "" || permalink == "." || strings.ContainsRune(permalink, 0) ||
		strings.Contains(permalink, "\\") {
		return "", "", ErrInvalid
	}

	parsedURL, err := url.Parse(permalink)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	if parsedURL.Scheme != "" || parsedURL.Host != "" || parsedURL.Opaque != "" ||
		parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return "", "", ErrInvalid
	}

	if !strings.HasPrefix(permalink, "/") {
		permalink = "/" + permalink
	}
	if hasParentSegment(permalink) {
		return "", "", ErrInvalid
	}

	cleanPermalink := path.Clean(permalink)
	if cleanPermalink == "/" {
		return "/", "index.html", nil
	}

	lastSegment := path.Base(cleanPermalink)
	if path.Ext(lastSegment) != "" {
		return cleanPermalink, strings.TrimPrefix(cleanPermalink, "/"), nil
	}

	return cleanPermalink + "/", strings.TrimPrefix(
		path.Join(cleanPermalink, "index.html"),
		"/",
	), nil
}

// FromPath converts a project-relative file path into a pretty permalink.
func FromPath(rawPath string, options PathOptions) (string, error) {
	sourcePath, err := cleanRelativePath(rawPath)
	if err != nil {
		return "", err
	}

	basePath := strings.TrimSpace(options.BasePath)
	if basePath != "" {
		basePath, err = cleanRelativePath(basePath)
		if err != nil {
			return "", err
		}
		if sourcePath == basePath {
			return "/", nil
		}
		if !strings.HasPrefix(sourcePath, basePath+"/") {
			return "", fmt.Errorf(
				"%w: path %q is not inside base path %q",
				ErrInvalid,
				rawPath,
				options.BasePath,
			)
		}

		sourcePath = strings.TrimPrefix(sourcePath, basePath+"/")
	}

	withoutExtension := strings.TrimSuffix(sourcePath, path.Ext(sourcePath))
	if withoutExtension == "" || withoutExtension == "." {
		return "/", nil
	}

	segments := strings.Split(withoutExtension, "/")
	if segments[len(segments)-1] == "index" {
		segments = segments[:len(segments)-1]
	}
	if len(segments) == 0 {
		return "/", nil
	}

	permalink, _, err := Normalize(path.Join(segments...))
	if err != nil {
		return "", err
	}

	return permalink, nil
}

// cleanRelativePath normalizes a slash-separated relative path.
func cleanRelativePath(rawPath string) (string, error) {
	cleanPath := strings.TrimSpace(strings.ReplaceAll(rawPath, "\\", "/"))
	if cleanPath == "" || cleanPath == "." || strings.ContainsRune(cleanPath, 0) ||
		path.IsAbs(cleanPath) {
		return "", ErrInvalid
	}

	for strings.HasPrefix(cleanPath, "./") {
		cleanPath = strings.TrimPrefix(cleanPath, "./")
	}
	if slices.Contains(strings.Split(cleanPath, "/"), "..") {
		return "", ErrInvalid
	}

	cleanPath = path.Clean(cleanPath)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", ErrInvalid
	}

	return cleanPath, nil
}

// hasParentSegment reports whether a slash-separated path contains a parent
// traversal segment.
func hasParentSegment(name string) bool {
	return slices.Contains(strings.Split(name, "/"), "..")
}
