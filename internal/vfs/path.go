package vfs

import (
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

func cleanPath(name string) (string, error) {
	rawName := strings.TrimSpace(name)
	if rawName == "" || strings.ContainsRune(rawName, 0) || filepath.VolumeName(rawName) != "" ||
		hasWindowsVolumeName(rawName) ||
		filepath.IsAbs(rawName) {
		return "", ErrPathInvalid
	}

	rawName = strings.ReplaceAll(rawName, "\\", "/")
	if path.IsAbs(rawName) {
		return "", ErrPathInvalid
	}

	if slices.Contains(strings.Split(rawName, "/"), "..") {
		return "", ErrPathInvalid
	}

	cleanName := path.Clean(rawName)
	if !fs.ValidPath(cleanName) {
		return "", ErrPathInvalid
	}

	return cleanName, nil
}

func hasWindowsVolumeName(name string) bool {
	return len(name) >= 2 && name[1] == ':' &&
		('A' <= name[0] && name[0] <= 'Z' || 'a' <= name[0] && name[0] <= 'z')
}

func cleanTopName(name string) (string, error) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return "", ErrTopDirInvalid
	}
	if cleanName == "." || strings.Contains(cleanName, "/") {
		return "", ErrTopDirInvalid
	}

	return cleanName, nil
}

func pathError(op, name string, err error) error {
	return &fs.PathError{Op: op, Path: name, Err: err}
}
