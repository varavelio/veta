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

// TestWriteMinifiesGeneratedHTMLOnly verifies HTML minification is extension-gated.
func TestWriteMinifiesGeneratedHTMLOnly(t *testing.T) {
	dir := t.TempDir()
	writer, err := New(dir, WithHTMLMinify(true))
	require.NoError(t, err)

	nonHTMLContent := map[string]string{
		"feed.xml":   "<feed>\n  <title> Keep XML spacing </title>\n</feed>\n",
		"notes.md":   "# Keep Markdown\n\nBody\n",
		"data.json":  "{\n  \"message\": \"keep json spacing\"\n}\n",
		"readme.txt": "hello\n  world\n",
		"app.js":     "function test() {\n  return true;\n}\n",
		"styles.css": "body {\n  color: red;\n}\n",
	}
	files := []File{
		{
			Content: []byte(
				"<!doctype html>\n<html>\n<body>\n  <div class=\"hero\"> Hello </div>\n</body>\n</html>\n",
			),
			Path: "index.HTML",
		},
	}
	for path, content := range nonHTMLContent {
		files = append(files, File{Content: []byte(content), Path: path})
	}

	require.NoError(t, writer.Write(files))

	index, err := os.ReadFile(filepath.Join(dir, "index.HTML"))
	require.NoError(t, err)
	require.NotContains(t, string(index), "\n")
	require.Contains(t, string(index), `<div class=hero>Hello</div>`)
	for path, want := range nonHTMLContent {
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		require.NoError(t, err)
		require.Equal(t, want, string(content))
	}
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

// TestWriteSiteDoesNotMinifyPublicHTML verifies public assets are copied as-is.
func TestWriteSiteDoesNotMinifyPublicHTML(t *testing.T) {
	dir := t.TempDir()
	writer, err := New(dir, WithHTMLMinify(true))
	require.NoError(t, err)

	publicHTML := "<html>\n  <body> Public HTML </body>\n</html>\n"
	require.NoError(t, writer.WriteSite(nil, fstest.MapFS{
		"public/public.html": {Data: []byte(publicHTML)},
	}))

	content, err := os.ReadFile(filepath.Join(dir, "public.html"))
	require.NoError(t, err)
	require.Equal(t, publicHTML, string(content))
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
