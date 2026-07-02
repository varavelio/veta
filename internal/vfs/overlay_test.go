package vfs

import (
	"errors"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestOverlayReadFileUsesHighestPriorityLayer(t *testing.T) {
	theme := fstest.MapFS{
		"templates/base.j2":       {Data: []byte("theme base")},
		"templates/theme-only.j2": {Data: []byte("theme only")},
	}
	project := fstest.MapFS{
		"templates/base.j2": {Data: []byte("project base")},
	}
	overlay := newTestOverlay(t, theme, project)

	content, err := fs.ReadFile(overlay, "templates/base.j2")
	require.NoError(t, err)
	require.Equal(t, "project base", string(content))

	content, err = fs.ReadFile(overlay, "templates/theme-only.j2")
	require.NoError(t, err)
	require.Equal(t, "theme only", string(content))

	source, ok := overlay.Source("templates/base.j2")
	require.True(t, ok)
	require.Equal(t, Source{Layer: "project", Path: "templates/base.j2"}, source)

	info, err := fs.Stat(overlay, "templates/base.j2")
	require.NoError(t, err)
	require.Equal(t, "base.j2", info.Name())
	require.False(t, info.IsDir())

	info, err = fs.Stat(overlay, "templates")
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestOverlayReadDirMergesDirectories(t *testing.T) {
	theme := fstest.MapFS{
		"templates/base.j2":       {Data: []byte("base")},
		"templates/shared.j2":     {Data: []byte("theme shared")},
		"templates/theme-only.j2": {Data: []byte("theme only")},
	}
	project := fstest.MapFS{
		"templates/project-only.j2": {Data: []byte("project only")},
		"templates/shared.j2":       {Data: []byte("project shared")},
	}
	overlay := newTestOverlay(t, theme, project)

	entries, err := fs.ReadDir(overlay, "templates")
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{"base.j2", "project-only.j2", "shared.j2", "theme-only.j2"},
		entryNames(entries),
	)

	content, err := fs.ReadFile(overlay, "templates/shared.j2")
	require.NoError(t, err)
	require.Equal(t, "project shared", string(content))
}

func TestOverlayOpenMergedDirectory(t *testing.T) {
	overlay := newTestOverlay(t,
		fstest.MapFS{"templates/a.j2": {Data: []byte("a")}},
		fstest.MapFS{"templates/b.j2": {Data: []byte("b")}},
	)

	file, err := overlay.Open("templates")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, file.Close())
	}()

	dirFile, ok := file.(fs.ReadDirFile)
	require.True(t, ok)

	first, err := dirFile.ReadDir(1)
	require.NoError(t, err)
	require.Equal(t, []string{"a.j2"}, entryNames(first))

	second, err := dirFile.ReadDir(1)
	require.NoError(t, err)
	require.Equal(t, []string{"b.j2"}, entryNames(second))

	third, err := dirFile.ReadDir(1)
	require.ErrorIs(t, err, io.EOF)
	require.Nil(t, third)
}

func TestOverlayFileBeatsLowerPriorityDirectory(t *testing.T) {
	theme := fstest.MapFS{"assets/logo.svg": {Data: []byte("logo")}}
	project := fstest.MapFS{"assets": {Data: []byte("file")}}
	overlay := newTestOverlay(t, theme, project)

	content, err := fs.ReadFile(overlay, "assets")
	require.NoError(t, err)
	require.Equal(t, "file", string(content))

	_, err = fs.ReadDir(overlay, "assets")
	require.Error(t, err)
	require.True(t, errors.Is(err, fs.ErrInvalid), "expected fs.ErrInvalid, got %v", err)
}

func TestOverlayWalkDir(t *testing.T) {
	overlay := newTestOverlay(t,
		fstest.MapFS{"templates/base.j2": {Data: []byte("base")}},
		fstest.MapFS{"components/ui/button.j2": {Data: []byte("button")}},
	)

	var paths []string
	err := fs.WalkDir(overlay, ".", func(path string, _ fs.DirEntry, err error) error {
		require.NoError(t, err)
		paths = append(paths, path)
		return nil
	})
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			".",
			"components",
			"components/ui",
			"components/ui/button.j2",
			"templates",
			"templates/base.j2",
		},
		paths,
	)
}

func TestOverlayErrors(t *testing.T) {
	_, err := NewOverlay()
	require.ErrorIs(t, err, ErrLayerRequired)

	_, err = NewOverlay(Layer{Name: "broken"})
	require.ErrorIs(t, err, ErrLayerInvalid)

	overlay := newTestOverlay(t, fstest.MapFS{}, fstest.MapFS{})

	_, err = fs.ReadFile(overlay, "missing.txt")
	require.ErrorIs(t, err, fs.ErrNotExist)

	for _, path := range []string{"", "../secret", "/absolute", `C:\secret`} {
		t.Run(path, func(t *testing.T) {
			_, err := fs.ReadFile(overlay, path)
			require.ErrorIs(t, err, ErrPathInvalid)
		})
	}

	_, ok := overlay.Source("missing.txt")
	require.False(t, ok)
}

func newTestOverlay(t *testing.T, theme, project fs.FS) *Overlay {
	t.Helper()

	overlay, err := NewOverlay(
		Layer{Name: "theme", FS: theme},
		Layer{Name: "project", FS: project},
	)
	require.NoError(t, err)

	return overlay
}

func entryNames(entries []fs.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names
}
