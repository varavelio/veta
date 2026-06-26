package output

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// TestWrite writes files and creates parent directories.
func TestWrite(t *testing.T) {
	dir := t.TempDir()
	writer, err := New(dir)
	require.NoError(t, err)

	require.NoError(t, writer.Write([]File{{Content: []byte("hello"), Path: "nested/page.html"}}))
	content, err := os.ReadFile(filepath.Join(dir, "nested", "page.html"))
	require.NoError(t, err)
	require.Equal(t, "hello", string(content))
}

// TestWriteClean removes previous output before writing.
func TestWriteClean(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "old.txt"), []byte("old"), 0o644))
	writer, err := New(dir, WithClean(true))
	require.NoError(t, err)

	require.NoError(t, writer.Write([]File{{Content: []byte("new"), Path: "new.txt"}}))
	_, err = os.Stat(filepath.Join(dir, "old.txt"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

// TestCopyPublic copies public files to the output root.
func TestCopyPublic(t *testing.T) {
	dir := t.TempDir()
	writer, err := New(dir)
	require.NoError(t, err)

	require.NoError(t, writer.CopyPublic(fstest.MapFS{
		"public/favicon.ico":    {Data: []byte("icon")},
		"public/assets/app.css": {Data: []byte("css")},
	}))

	content, err := os.ReadFile(filepath.Join(dir, "favicon.ico"))
	require.NoError(t, err)
	require.Equal(t, "icon", string(content))
	content, err = os.ReadFile(filepath.Join(dir, "assets", "app.css"))
	require.NoError(t, err)
	require.Equal(t, "css", string(content))
}

// TestWriteSiteDetectsCollisions verifies document and public collision checks.
func TestWriteSiteDetectsCollisions(t *testing.T) {
	writer, err := New(t.TempDir())
	require.NoError(t, err)

	err = writer.WriteSite([]File{{Content: []byte("doc"), Path: "app.css"}}, fstest.MapFS{
		"public/app.css": {Data: []byte("public")},
	})
	require.ErrorIs(t, err, ErrPathDuplicate)
}

// TestWriteSiteIgnoresMissingPublic verifies missing public directories are ok.
func TestWriteSiteIgnoresMissingPublic(t *testing.T) {
	dir := t.TempDir()
	writer, err := New(dir)
	require.NoError(t, err)

	require.NoError(
		t,
		writer.WriteSite([]File{{Content: []byte("doc"), Path: "index.html"}}, fstest.MapFS{}),
	)
	content, err := os.ReadFile(filepath.Join(dir, "index.html"))
	require.NoError(t, err)
	require.Equal(t, "doc", string(content))
}

// TestWriterErrors verifies path and directory validation.
func TestWriterErrors(t *testing.T) {
	_, err := New("")
	require.ErrorIs(t, err, ErrDirInvalid)

	writer, err := New(t.TempDir())
	require.NoError(t, err)

	tests := []string{"", ".", "/absolute.html", "../escape.html", `C:\escape.html`}
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			err := writer.Write([]File{{Content: []byte("bad"), Path: test}})
			require.ErrorIs(t, err, ErrPathInvalid)
		})
	}

	err = writer.Write([]File{{Path: "same.txt"}, {Path: "same.txt"}})
	require.ErrorIs(t, err, ErrPathDuplicate)
}
