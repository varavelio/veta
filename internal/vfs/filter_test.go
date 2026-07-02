package vfs

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestAllowTopDirs(t *testing.T) {
	theme := fstest.MapFS{
		"README.md":                 {Data: []byte("ignored")},
		"components/youtube.j2":     {Data: []byte("youtube")},
		"data/theme.json":           {Data: []byte("{}")},
		"pages/malicious.js":        {Data: []byte("ignored")},
		"public/app.css":            {Data: []byte("body{}")},
		"templates/layouts/base.j2": {Data: []byte("base")},
	}
	filtered, err := AllowTopDirs(theme, "templates", "components", "filters", "data", "public")
	require.NoError(t, err)

	content, err := fs.ReadFile(filtered, "templates/layouts/base.j2")
	require.NoError(t, err)
	require.Equal(t, "base", string(content))

	entries, err := fs.ReadDir(filtered, ".")
	require.NoError(t, err)
	require.Equal(t, []string{"components", "data", "public", "templates"}, entryNames(entries))

	_, err = fs.ReadFile(filtered, "pages/malicious.js")
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = fs.ReadFile(filtered, "README.md")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAllowTopDirsWorksWithOverlay(t *testing.T) {
	theme := fstest.MapFS{
		"pages/ignored.js":        {Data: []byte("ignored")},
		"templates/base.j2":       {Data: []byte("theme")},
		"templates/theme-only.j2": {Data: []byte("theme only")},
	}
	filteredTheme, err := AllowTopDirs(theme, "templates")
	require.NoError(t, err)

	overlay := newTestOverlay(t, filteredTheme, fstest.MapFS{
		"pages/site.js":     {Data: []byte("allowed locally")},
		"templates/base.j2": {Data: []byte("project")},
	})

	content, err := fs.ReadFile(overlay, "templates/base.j2")
	require.NoError(t, err)
	require.Equal(t, "project", string(content))

	content, err = fs.ReadFile(overlay, "templates/theme-only.j2")
	require.NoError(t, err)
	require.Equal(t, "theme only", string(content))

	content, err = fs.ReadFile(overlay, "pages/site.js")
	require.NoError(t, err)
	require.Equal(t, "allowed locally", string(content))

	_, err = fs.ReadFile(overlay, "pages/ignored.js")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAllowTopDirsErrors(t *testing.T) {
	_, err := AllowTopDirs(nil, "templates")
	require.ErrorIs(t, err, ErrFSRequired)

	_, err = AllowTopDirs(fstest.MapFS{}, "templates/nested")
	require.ErrorIs(t, err, ErrTopDirInvalid)

	filtered, err := AllowTopDirs(fstest.MapFS{}, "templates")
	require.NoError(t, err)

	_, err = fs.ReadDir(filtered, "../templates")
	require.True(t, errors.Is(err, ErrPathInvalid), "expected ErrPathInvalid, got %v", err)
}
