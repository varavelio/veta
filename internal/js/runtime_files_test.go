package js

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunnerFileAndParseAPIs(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "content"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "data"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "data", "site.json"),
		[]byte(`{"name":"Veta","count":2}`),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "data", "navigation.yaml"),
		[]byte("items:\n  - label: Docs\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "data", "theme.toml"),
		[]byte("name = \"Clean\"\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "content", "post.md"),
		[]byte("---\ntitle: Hello\n---\n\n# Body\n"),
		0o644,
	))

	result, err := New(WithRoot(root)).ExecuteString("site.js", `
		export default function({ files, parse }) {
			const site = parse.json(files.readFile("data/site.json"));
			const navigation = parse.yaml(files.readFile("data/navigation.yaml"));
			const theme = parse.toml(files.readFile("data/theme.toml"));
			const post = parse.markdown(files.readFile("content/post.md"));
			return {
				count: site.count,
				files: files.listFiles("data/*"),
				label: navigation.items[0].label,
				permalink: files.toPermalink("content/post.md", { stripPrefix: "content" }),
				postBody: post.content,
				postTitle: post.frontmatter.title,
				theme: theme.name,
			};
		}
	`)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, result.ExportTo(&got))
	require.Equal(t, map[string]any{
		"count":     int64(2),
		"files":     []string{"data/navigation.yaml", "data/site.json", "data/theme.toml"},
		"label":     "Docs",
		"permalink": "/post/",
		"postBody":  "# Body\n",
		"postTitle": "Hello",
		"theme":     "Clean",
	}, got)
}

func TestRunnerParseAPIErrors(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "parse json requires string",
			code: `export default function({ parse }) { return parse.json(123); }`,
			want: "parse.json content must be a string",
		},
		{
			name: "parse json reports invalid content",
			code: `export default function({ parse }) { return parse.json("{"); }`,
			want: "parse json",
		},
		{
			name: "parse markdown reports invalid frontmatter",
			code: `export default function({ parse }) { return parse.markdown("---\ntitle: [broken\n---\nBody\n"); }`,
			want: "parse markdown",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New().ExecuteString(test.name+".js", test.code)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}

func TestRunnerReadFileErrors(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "non string path",
			code: `export default function({ files }) { return files.readFile(123); }`,
			want: "files.readFile path must be a string",
		},
		{
			name: "absolute path",
			code: `export default function({ files }) { return files.readFile("/content/post.md"); }`,
			want: ErrPathOutsideRoot.Error(),
		},
		{
			name: "outside root path",
			code: `export default function({ files }) { return files.readFile("../post.md"); }`,
			want: ErrPathOutsideRoot.Error(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New().ExecuteString(test.name+".js", test.code)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}
