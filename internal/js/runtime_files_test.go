package js

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestParseMarkdownDocument verifies Markdown front matter parsing.
func TestParseMarkdownDocument(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		wantContent     string
		wantFrontMatter map[string]any
	}{
		{
			name:            "without front matter",
			content:         "# Home\n\nBody.\n",
			wantContent:     "# Home\n\nBody.\n",
			wantFrontMatter: map[string]any{},
		},
		{
			name:            "delimiter after first line is body content",
			content:         "\n---\ntitle: Late\n---\nBody\n",
			wantContent:     "\n---\ntitle: Late\n---\nBody\n",
			wantFrontMatter: map[string]any{},
		},
		{
			name:            "empty yaml front matter",
			content:         "---\n---\nBody\n",
			wantContent:     "Body\n",
			wantFrontMatter: map[string]any{},
		},
		{
			name:        "yaml front matter",
			content:     "---\ntitle: Hello\ndraft: false\ncount: 2\ntags:\n  - go\n  - ssg\n---\n\n# Hello\n\nBody.\n",
			wantContent: "# Hello\n\nBody.\n",
			wantFrontMatter: map[string]any{
				"count": int64(2),
				"draft": false,
				"tags":  []any{"go", "ssg"},
				"title": "Hello",
			},
		},
		{
			name:        "toml front matter with windows line endings",
			content:     "+++\r\ntitle = \"Release\"\r\nweight = 3\r\ndraft = false\r\npublished = 2026-06-30T12:34:56Z\r\ntags = [\"go\", \"toml\"]\r\n\r\n[meta]\r\nauthor = \"Veta\"\r\n+++\r\n\r\n# Release\r\n\r\nBody.\r\n",
			wantContent: "# Release\r\n\r\nBody.\r\n",
			wantFrontMatter: map[string]any{
				"draft":     false,
				"meta":      map[string]any{"author": "Veta"},
				"published": "2026-06-30T12:34:56Z",
				"tags":      []any{"go", "toml"},
				"title":     "Release",
				"weight":    int64(3),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := parseMarkdownDocument(test.content)
			require.NoError(t, err)
			require.Equal(t, test.wantContent, document["content"])
			require.Equal(t, test.wantFrontMatter, document["frontmatter"])
		})
	}
}

// TestParseMarkdownDocumentErrors verifies front matter parse failures.
func TestParseMarkdownDocumentErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "unterminated yaml front matter",
			content: "---\ntitle: Missing close\nBody\n",
			want:    "unterminated front matter block",
		},
		{
			name:    "unterminated toml front matter",
			content: "+++\ntitle = \"Missing close\"\nBody\n",
			want:    "unterminated front matter block",
		},
		{
			name:    "yaml front matter must be object",
			content: "---\n- one\n- two\n---\nBody\n",
			want:    "front matter must be an object",
		},
		{
			name:    "malformed yaml front matter",
			content: "---\ntitle: [broken\n---\nBody\n",
			want:    "decode yaml",
		},
		{
			name:    "malformed toml front matter",
			content: "+++\ntitle = \n+++\nBody\n",
			want:    "decode toml",
		},
		{
			name:    "yaml non finite number",
			content: "---\nvalue: .inf\n---\nBody\n",
			want:    "$.value has non-finite number",
		},
		{
			name:    "toml non finite number",
			content: "+++\nvalue = nan\n+++\nBody\n",
			want:    "$.value has non-finite number",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseMarkdownDocument(test.content)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}

// TestParseYAMLValueRejectsMultipleDocuments verifies YAML document boundaries.
func TestParseYAMLValueRejectsMultipleDocuments(t *testing.T) {
	_, err := parseYAMLValue([]byte("name: one\n---\nname: two\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple yaml documents are not supported")
}

// TestNormalizeStructuredValue verifies conversion to JavaScript-safe values.
func TestNormalizeStructuredValue(t *testing.T) {
	instant := time.Date(2026, 6, 30, 12, 0, 0, 123, time.UTC)

	value, err := normalizeStructuredValue(map[any]any{
		"nested": map[any]any{
			"items": []any{
				json.Number("2"),
				json.Number("3.5"),
				instant,
				uint(7),
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"nested": map[string]any{
			"items": []any{
				int64(2),
				3.5,
				"2026-06-30T12:00:00.000000123Z",
				uint64(7),
			},
		},
	}, value)
}

// TestNormalizeStructuredValueErrors verifies rejected structured values.
func TestNormalizeStructuredValueErrors(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "non string map key",
			value: map[any]any{1: "bad"},
			want:  "$ has non-string map key",
		},
		{
			name:  "non finite float",
			value: map[string]any{"value": math.Inf(1)},
			want:  "$.value has non-finite number",
		},
		{
			name:  "invalid json number",
			value: json.Number("bad"),
			want:  "$ has invalid number \"bad\"",
		},
		{
			name:  "unsupported value type",
			value: struct{}{},
			want:  "$ has unsupported value type",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeStructuredValue(test.value)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}

// TestRunnerReadMarkdownFileParsesTOMLFrontMatter verifies JavaScript API output.
func TestRunnerReadMarkdownFileParsesTOMLFrontMatter(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "content"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "content", "post.md"),
		[]byte(
			"+++\ntitle = \"TOML Post\"\ndraft = false\ntags = [\"toml\", \"frontmatter\"]\n+++\n\n# TOML Post\n",
		),
		0o644,
	))

	result, err := New(WithRoot(root)).ExecuteString("markdown.js", `
		export default function({ files }) {
			const post = files.readMarkdownFile("./content/post.md");
			return {
				content: post.content,
				draft: post.frontmatter.draft,
				path: post.path,
				tags: post.frontmatter.tags,
				title: post.frontmatter.title,
			};
		}
	`)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, result.ExportTo(&got))
	require.Equal(t, map[string]any{
		"content": "# TOML Post\n",
		"draft":   false,
		"path":    "content/post.md",
		"tags":    []any{"toml", "frontmatter"},
		"title":   "TOML Post",
	}, got)
}

// TestRunnerReadMarkdownFileErrors verifies JavaScript API error context.
func TestRunnerReadMarkdownFileErrors(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "content"), 0o755))
	files := map[string]string{
		"bad-yaml.md":     "---\ntitle: [broken\n---\nBody\n",
		"bad-toml.md":     "+++\ntitle = \n+++\nBody\n",
		"scalar.md":       "---\n42\n---\nBody\n",
		"unterminated.md": "---\ntitle: Missing close\nBody\n",
	}
	for name, content := range files {
		require.NoError(
			t,
			os.WriteFile(filepath.Join(root, "content", name), []byte(content), 0o644),
		)
	}

	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "non string path",
			code: `
				export default function({ files }) {
					return files.readMarkdownFile(123);
				}
			`,
			want: "files.readMarkdownFile path must be a string",
		},
		{
			name: "absolute path",
			code: `
				export default function({ files }) {
					return files.readMarkdownFile("/content/post.md");
				}
			`,
			want: ErrPathOutsideRoot.Error(),
		},
		{
			name: "outside root path",
			code: `
				export default function({ files }) {
					return files.readMarkdownFile("../post.md");
				}
			`,
			want: ErrPathOutsideRoot.Error(),
		},
		{
			name: "unterminated front matter",
			code: `
				export default function({ files }) {
					return files.readMarkdownFile("content/unterminated.md");
				}
			`,
			want: "parse markdown file content/unterminated.md: unterminated front matter block",
		},
		{
			name: "front matter must be object",
			code: `
				export default function({ files }) {
					return files.readMarkdownFile("content/scalar.md");
				}
			`,
			want: "parse markdown file content/scalar.md: front matter must be an object",
		},
		{
			name: "malformed yaml",
			code: `
				export default function({ files }) {
					return files.readMarkdownFile("content/bad-yaml.md");
				}
			`,
			want: "parse markdown file content/bad-yaml.md: decode yaml",
		},
		{
			name: "malformed toml",
			code: `
				export default function({ files }) {
					return files.readMarkdownFile("content/bad-toml.md");
				}
			`,
			want: "parse markdown file content/bad-toml.md: decode toml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(WithRoot(root)).ExecuteString(test.name+".js", test.code)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}
