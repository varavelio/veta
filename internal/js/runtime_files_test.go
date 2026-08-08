package js

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/veta/internal/markdown"
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

	result, err := New(
		WithRoot(root),
		WithMarkdownRenderer(markdown.New()),
	).ExecuteString("site.js", `
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
				postHTML: post.html,
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
		"postHTML":  "<h1>Body</h1>\n",
		"postTitle": "Hello",
		"theme":     "Clean",
	}, got)
}

func TestRunnerFileAPIsWithFiles(t *testing.T) {
	files := fstest.MapFS{
		"content/index.md":                &fstest.MapFile{Data: []byte("# Home\n")},
		"templates/vara/icons/check.svg":  &fstest.MapFile{Data: []byte("<svg>check</svg>")},
		"templates/vara/icons/github.svg": &fstest.MapFile{Data: []byte("<svg>github</svg>")},
		"templates/vara/icons/icons.json": &fstest.MapFile{
			Data: []byte(`{"icons":[{"name":"check"}]}`),
		},
	}

	result, err := New(
		WithFiles(files),
		WithMarkdownRenderer(markdown.New()),
	).ExecuteString("overlay.js", `
		export default function({ files }) {
			return {
				content: files.readFile("content/index.md"),
				icons: files.listFiles("templates/vara/icons/*.svg"),
				json: files.readFile("templates/vara/icons/icons.json"),
			};
		}
	`)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, result.ExportTo(&got))
	require.Equal(t, map[string]any{
		"content": "# Home\n",
		"icons":   []string{"templates/vara/icons/check.svg", "templates/vara/icons/github.svg"},
		"json":    `{"icons":[{"name":"check"}]}`,
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
