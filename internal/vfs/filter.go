package vfs

import (
	"fmt"
	"io/fs"
	"sort"
)

type allowFS struct {
	allowed map[string]struct{}
	files   fs.FS
}

// AllowTopDirs hides every root entry except the provided top-level names.
func AllowTopDirs(files fs.FS, names ...string) (fs.FS, error) {
	if files == nil {
		return nil, ErrFSRequired
	}

	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		cleanName, err := cleanTopName(name)
		if err != nil {
			return nil, fmt.Errorf("%w: %q", ErrTopDirInvalid, name)
		}

		allowed[cleanName] = struct{}{}
	}

	return &allowFS{allowed: allowed, files: files}, nil
}

func (files *allowFS) Open(name string) (fs.File, error) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return nil, pathError("open", name, err)
	}

	if cleanName == "." {
		info, err := fs.Stat(files.files, ".")
		if err != nil {
			return nil, err
		}
		entries, err := files.ReadDir(".")
		if err != nil {
			return nil, err
		}

		return newDirFile(cleanName, info, entries), nil
	}

	if !files.isAllowed(cleanName) {
		return nil, pathError("open", cleanName, fs.ErrNotExist)
	}

	return files.files.Open(cleanName)
}

func (files *allowFS) ReadFile(name string) ([]byte, error) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return nil, pathError("read", name, err)
	}
	if !files.isAllowed(cleanName) || cleanName == "." {
		return nil, pathError("read", cleanName, fs.ErrNotExist)
	}

	return fs.ReadFile(files.files, cleanName)
}

func (files *allowFS) ReadDir(name string) ([]fs.DirEntry, error) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return nil, pathError("readdir", name, err)
	}

	if cleanName != "." {
		if !files.isAllowed(cleanName) {
			return nil, pathError("readdir", cleanName, fs.ErrNotExist)
		}

		return fs.ReadDir(files.files, cleanName)
	}

	rootEntries, err := fs.ReadDir(files.files, ".")
	if err != nil {
		return nil, err
	}

	entries := make([]fs.DirEntry, 0, len(rootEntries))
	for _, entry := range rootEntries {
		if _, ok := files.allowed[entry.Name()]; ok {
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Name() < entries[right].Name()
	})

	return entries, nil
}

func (files *allowFS) Stat(name string) (fs.FileInfo, error) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return nil, pathError("stat", name, err)
	}
	if cleanName != "." && !files.isAllowed(cleanName) {
		return nil, pathError("stat", cleanName, fs.ErrNotExist)
	}

	return fs.Stat(files.files, cleanName)
}

func (files *allowFS) isAllowed(name string) bool {
	if name == "." {
		return true
	}

	topName := name
	for index, char := range name {
		if char == '/' {
			topName = name[:index]
			break
		}
	}

	_, ok := files.allowed[topName]
	return ok
}
