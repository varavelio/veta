package view

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
)

type templateLoader struct {
	extensions []string
	files      fs.FS
}

func (loader *templateLoader) Abs(base string, name string) string {
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

	if path.Ext(cleanName) != "" {
		if exists, err := fileExists(loader.files, cleanName); err != nil || !exists {
			if err != nil {
				return "", err
			}

			return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, cleanName)
		}

		return cleanName, nil
	}

	matches := make([]string, 0, 1)
	for _, extension := range loader.extensions {
		candidate := cleanName + extension
		exists, err := fileExists(loader.files, candidate)
		if err != nil {
			return "", err
		}
		if exists {
			matches = append(matches, candidate)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, cleanName)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%w: %s matches %s", ErrTemplateAmbiguous, cleanName, strings.Join(matches, ", "))
	}
}

func cleanTemplateName(name string) (string, error) {
	rawName := strings.TrimSpace(normalizePathSeparators(name))
	if rawName == "" || strings.ContainsRune(rawName, 0) || path.IsAbs(rawName) {
		return "", ErrTemplateNameInvalid
	}

	for _, segment := range strings.Split(rawName, "/") {
		if segment == ".." {
			return "", ErrTemplateNameInvalid
		}
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

func normalizePathSeparators(name string) string {
	return strings.ReplaceAll(name, "\\", "/")
}
