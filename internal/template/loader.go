package template

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"
)

type templateLoader struct {
	files fs.FS
}

func (loader *templateLoader) Abs(base, name string) string {
	name = normalizePathSeparators(name)
	if strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../") {
		base = normalizePathSeparators(base)
		baseDir := path.Dir(base)
		if baseDir == "." {
			return path.Clean(name)
		}

		return path.Clean(path.Join(baseDir, name))
	}

	return name
}

func (loader *templateLoader) Get(name string) (io.Reader, error) {
	resolvedName, err := loader.resolve(name)
	if err != nil {
		return nil, err
	}

	content, err := fs.ReadFile(loader.files, resolvedName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrTemplateNotFound, name)
		}

		return nil, fmt.Errorf("read template %s: %w", resolvedName, err)
	}

	return bytes.NewReader(content), nil
}

func (loader *templateLoader) resolve(name string) (string, error) {
	cleanName, err := cleanTemplateName(name)
	if err != nil {
		return "", err
	}

	if templatePathIgnored(cleanName) {
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, cleanName)
	}

	if path.Ext(cleanName) != "" {
		exists, err := fileExists(loader.files, cleanName)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, cleanName)
		}

		return cleanName, nil
	}

	matches, err := loader.resolveExtensionless(cleanName)
	if err != nil {
		return "", err
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, cleanName)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf(
			"%w: %s matches %s",
			ErrTemplateAmbiguous,
			cleanName,
			strings.Join(matches, ", "),
		)
	}
}

// resolveExtensionless finds valid template files with a matching stem.
func (loader *templateLoader) resolveExtensionless(cleanName string) ([]string, error) {
	directory, stem := path.Split(cleanName)
	directory = strings.TrimSuffix(directory, "/")
	if directory == "" {
		directory = "."
	}

	entries, err := fs.ReadDir(loader.files, directory)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read template directory %s: %w", directory, err)
	}

	matches := make([]string, 0, 1)
	for _, entry := range entries {
		if entry.IsDir() || templateFileNameIgnored(entry.Name()) {
			continue
		}
		if fileStem(entry.Name()) != stem {
			continue
		}

		matches = append(matches, path.Join(directory, entry.Name()))
	}
	sort.Strings(matches)

	return matches, nil
}

func cleanTemplateName(name string) (string, error) {
	rawName := strings.TrimSpace(normalizePathSeparators(name))
	if rawName == "" || strings.ContainsRune(rawName, 0) || path.IsAbs(rawName) {
		return "", ErrTemplateNameInvalid
	}

	if slices.Contains(strings.Split(rawName, "/"), "..") {
		return "", ErrTemplateNameInvalid
	}

	cleanName := path.Clean(rawName)
	if cleanName == "." || !fs.ValidPath(cleanName) {
		return "", ErrTemplateNameInvalid
	}

	return cleanName, nil
}

func fileExists(files fs.FS, name string) (bool, error) {
	info, err := fs.Stat(files, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("stat template %s: %w", name, err)
	}

	return !info.IsDir(), nil
}

// fileStem returns the file name without its final extension.
func fileStem(name string) string {
	return strings.TrimSuffix(name, path.Ext(name))
}

// templatePathIgnored reports whether a path contains an ignored file segment.
func templatePathIgnored(name string) bool {
	return slices.ContainsFunc(strings.Split(name, "/"), templateFileNameIgnored)
}

// templateFileNameIgnored reports whether a template file should be skipped.
func templateFileNameIgnored(name string) bool {
	lowerName := strings.ToLower(name)

	return strings.HasPrefix(name, ".") ||
		strings.HasSuffix(name, "~") ||
		strings.HasSuffix(lowerName, ".tmp")
}

func normalizePathSeparators(name string) string {
	return strings.ReplaceAll(name, "\\", "/")
}
