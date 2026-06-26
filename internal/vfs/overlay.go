package vfs

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
)

// Layer is one filesystem layer in an overlay. Later layers passed to
// NewOverlay have higher priority than earlier layers.
type Layer struct {
	// Name identifies this layer in diagnostics and Source results.
	Name string

	// FS contains this layer's files.
	FS fs.FS
}

// Source identifies which overlay layer provides a path.
type Source struct {
	Layer string
	Path  string
}

// Overlay merges multiple filesystems into one read-only filesystem.
type Overlay struct {
	layers []Layer
}

// NewOverlay creates a read-only filesystem where later layers override earlier
// layers. Directories with the same path are merged.
func NewOverlay(layers ...Layer) (*Overlay, error) {
	if len(layers) == 0 {
		return nil, ErrLayerRequired
	}

	clone := make([]Layer, len(layers))
	for index, layer := range layers {
		if layer.FS == nil {
			return nil, fmt.Errorf("%w: layer %d has no filesystem", ErrLayerInvalid, index)
		}
		if layer.Name == "" {
			layer.Name = fmt.Sprintf("layer-%d", index+1)
		}

		clone[index] = layer
	}

	return &Overlay{layers: clone}, nil
}

// Open opens a file or a merged directory from the overlay.
func (overlay *Overlay) Open(name string) (fs.File, error) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return nil, pathError("open", name, err)
	}

	layerIndex, info, ok, err := overlay.find(cleanName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, pathError("open", cleanName, fs.ErrNotExist)
	}

	if !info.IsDir() {
		file, err := overlay.layers[layerIndex].FS.Open(cleanName)
		if err != nil {
			return nil, fmt.Errorf("open %s from layer %s: %w", cleanName, overlay.layers[layerIndex].Name, err)
		}

		return file, nil
	}

	entries, err := overlay.readMergedDir(cleanName)
	if err != nil {
		return nil, err
	}

	return newDirFile(cleanName, info, entries), nil
}

// ReadFile reads a file from the highest-priority layer that contains it.
func (overlay *Overlay) ReadFile(name string) ([]byte, error) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return nil, pathError("read", name, err)
	}

	layerIndex, info, ok, err := overlay.find(cleanName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, pathError("read", cleanName, fs.ErrNotExist)
	}
	if info.IsDir() {
		return nil, pathError("read", cleanName, fs.ErrInvalid)
	}

	content, err := fs.ReadFile(overlay.layers[layerIndex].FS, cleanName)
	if err != nil {
		return nil, fmt.Errorf("read %s from layer %s: %w", cleanName, overlay.layers[layerIndex].Name, err)
	}

	return content, nil
}

// ReadDir returns merged directory entries from every layer containing name as a
// directory. Higher-priority layers win entry name conflicts.
func (overlay *Overlay) ReadDir(name string) ([]fs.DirEntry, error) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return nil, pathError("readdir", name, err)
	}

	_, info, ok, err := overlay.find(cleanName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, pathError("readdir", cleanName, fs.ErrNotExist)
	}
	if !info.IsDir() {
		return nil, pathError("readdir", cleanName, fs.ErrInvalid)
	}

	return overlay.readMergedDir(cleanName)
}

// Source reports the highest-priority layer that provides name.
func (overlay *Overlay) Source(name string) (Source, bool) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return Source{}, false
	}

	layerIndex, _, ok, err := overlay.find(cleanName)
	if err != nil || !ok {
		return Source{}, false
	}

	return Source{Layer: overlay.layers[layerIndex].Name, Path: cleanName}, true
}

// Stat returns file information for the highest-priority layer that contains
// name.
func (overlay *Overlay) Stat(name string) (fs.FileInfo, error) {
	cleanName, err := cleanPath(name)
	if err != nil {
		return nil, pathError("stat", name, err)
	}

	_, info, ok, err := overlay.find(cleanName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, pathError("stat", cleanName, fs.ErrNotExist)
	}

	return info, nil
}

func (overlay *Overlay) find(name string) (int, fs.FileInfo, bool, error) {
	for index := len(overlay.layers) - 1; index >= 0; index-- {
		info, err := fs.Stat(overlay.layers[index].FS, name)
		if err == nil {
			return index, info, true, nil
		}
		if !isNotExist(err) {
			return 0, nil, false, fmt.Errorf("stat %s from layer %s: %w", name, overlay.layers[index].Name, err)
		}
	}

	return 0, nil, false, nil
}

func (overlay *Overlay) readMergedDir(name string) ([]fs.DirEntry, error) {
	entriesByName := map[string]fs.DirEntry{}
	for _, layer := range overlay.layers {
		entries, err := fs.ReadDir(layer.FS, name)
		if err != nil {
			if isNotExist(err) || isInvalid(err) {
				continue
			}

			return nil, fmt.Errorf("read directory %s from layer %s: %w", name, layer.Name, err)
		}

		for _, entry := range entries {
			entriesByName[entry.Name()] = entry
		}
	}

	names := make([]string, 0, len(entriesByName))
	for name := range entriesByName {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, entriesByName[name])
	}

	return entries, nil
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

func isInvalid(err error) bool {
	return errors.Is(err, fs.ErrInvalid)
}
