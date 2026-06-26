package pages

import (
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"
)

// normalizePermalink converts a user permalink into a canonical permalink and
// relative output file path.
func normalizePermalink(rawPermalink string) (string, string, error) {
	permalink := strings.TrimSpace(rawPermalink)
	if permalink == "" || permalink == "." || strings.ContainsRune(permalink, 0) ||
		strings.Contains(permalink, "\\") {
		return "", "", ErrPermalinkInvalid
	}

	parsedURL, err := url.Parse(permalink)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrPermalinkInvalid, err)
	}
	if parsedURL.Scheme != "" || parsedURL.Host != "" || parsedURL.Opaque != "" ||
		parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return "", "", ErrPermalinkInvalid
	}

	if !strings.HasPrefix(permalink, "/") {
		permalink = "/" + permalink
	}
	if hasParentSegment(permalink) {
		return "", "", ErrPermalinkInvalid
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

// hasParentSegment reports whether a slash-separated path contains a parent
// traversal segment.
func hasParentSegment(name string) bool {
	return slices.Contains(strings.Split(name, "/"), "..")
}
