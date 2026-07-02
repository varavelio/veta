package template

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestRendererRender(t *testing.T) {
	tests := []struct {
		name    string
		render  string
		context any
		want    string
	}{
		{
			name:   "renders arbitrary template extension",
			render: "pages/home",
			context: Context{
				"page": map[string]any{"title": "Home"},
			},
			want: "<h1>Home</h1>",
		},
		{
			name:   "renders extensionless template from nested directory",
			render: "emails/welcome",
			context: map[string]string{
				"name": "Veta",
			},
			want: "Welcome Veta",
		},
		{
			name:    "renders explicit extension",
			render:  "plain.txt",
			context: nil,
			want:    "plain text",
		},
	}

	files := fstest.MapFS{
		"emails/welcome.njk": {Data: []byte("Welcome {{ name }}")},
		"pages/home.twig":    {Data: []byte("<h1>{{ page.title }}</h1>")},
		"plain.txt":          {Data: []byte("plain text")},
	}
	renderer, err := New(files)
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := renderer.Render(test.render, test.context)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestRendererLayoutsAndIncludes(t *testing.T) {
	files := fstest.MapFS{
		"layouts/base.j2": {Data: []byte(strings.Join([]string{
			"<html>",
			"<head><title>{{ page.title }}</title></head>",
			"<body>",
			"{% block content %}{% endblock %}",
			"{% include \"partials/footer\" %}",
			"</body>",
			"</html>",
		}, ""))},
		"pages/home.j2": {Data: []byte(strings.Join([]string{
			"{% extends \"layouts/base\" %}",
			"{% block content %}<main>{{ page.content }}</main>{% endblock %}",
		}, ""))},
		"partials/footer.html": {Data: []byte("<footer>{{ data.site.title }}</footer>")},
	}
	renderer, err := New(files)
	require.NoError(t, err)

	got, err := renderer.Render("pages/home", Context{
		"data": map[string]any{
			"site": map[string]any{"title": "Veta"},
		},
		"page": map[string]any{
			"content": "Hello",
			"title":   "Home",
		},
	})
	require.NoError(t, err)
	require.Equal(
		t,
		"<html><head><title>Home</title></head><body><main>Hello</main><footer>Veta</footer></body></html>",
		got,
	)
}

func TestRendererFilters(t *testing.T) {
	files := fstest.MapFS{
		"page.j2": {Data: []byte(`{{ page.title|surround:"!" }} {{ page.content|trusted }}`)},
	}
	renderer, err := New(files,
		WithFilter("surround", func(input, parameter any) (any, error) {
			return fmt.Sprintf("%s%s%s", parameter, input, parameter), nil
		}),
		WithFilter("trusted", func(input, _ any) (any, error) {
			return SafeString(fmt.Sprint(input)), nil
		}),
	)
	require.NoError(t, err)

	got, err := renderer.Render("page", Context{
		"page": map[string]any{
			"content": "<strong>safe</strong>",
			"title":   "Veta",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "!Veta! <strong>safe</strong>", got)
}

func TestRendererGlobals(t *testing.T) {
	files := fstest.MapFS{
		"page.j2": {Data: []byte(`{{ greet(page.title) }}`)},
	}
	renderer, err := New(files, WithGlobal("greet", func(value string) string {
		return "Hello " + value
	}))
	require.NoError(t, err)

	got, err := renderer.Render("page", Context{"page": map[string]any{"title": "Veta"}})
	require.NoError(t, err)
	require.Equal(t, "Hello Veta", got)
}

func TestRendererRegexReplaceGlobal(t *testing.T) {
	files := fstest.MapFS{
		"page.j2": {Data: []byte(strings.Join([]string{
			`{% set slug = regex_replace(page.title, "[^a-zA-Z0-9]+", "-") %}`,
			`{{ slug }} {{ regex_replace("World Hello", "(\\w+) (\\w+)", "$2 $1") }}`,
		}, ""))},
	}
	renderer, err := New(files, WithRegexReplace())
	require.NoError(t, err)

	got, err := renderer.Render("page", Context{"page": map[string]any{"title": "Hello, Veta!"}})
	require.NoError(t, err)
	require.Equal(t, "Hello-Veta- Hello World", got)
}

func TestRendererBase64Filters(t *testing.T) {
	files := fstest.MapFS{
		"page.j2": {Data: []byte(`{{ "hello"|base64_encode }} {{ "aGVsbG8="|base64_decode }}`)},
	}
	renderer, err := New(files,
		WithFilter("base64_encode", func(input, _ any) (any, error) {
			return "aGVsbG8=", nil
		}),
		WithFilter("base64_decode", func(input, _ any) (any, error) {
			return "hello", nil
		}),
	)
	require.NoError(t, err)

	got, err := renderer.Render("page", nil)
	require.NoError(t, err)
	require.Equal(t, "aGVsbG8= hello", got)
}

func TestRendererLoadData(t *testing.T) {
	files := fstest.MapFS{
		"page.j2": {Data: []byte(strings.Join([]string{
			`{% set site = load_data("data/site.json")|parse_json %}`,
			`{% set raw = load_data("data/raw.txt") %}`,
			`{% set remote = load_data("https://example.test/data.json")|parse_json %}`,
			`{% set site_again = load_data("data/site.json")|parse_json %}`,
			`{{ site.name }} {{ raw }} {{ site_again.name }} {{ remote.timeout }}`,
		}, ""))},
	}
	renderer, err := New(files, WithLoadData(func(request LoadDataRequest) (any, error) {
		switch {
		case request.Path == "data/site.json":
			return `{"name":"Veta"}`, nil
		case request.Path == "data/raw.txt":
			return "Raw", nil
		case request.URL == "https://example.test/data.json":
			return `{"timeout":"OK"}`, nil
		default:
			return nil, fmt.Errorf("unexpected request %+v", request)
		}
	}), WithFilter("parse_json", func(input, _ any) (any, error) {
		text, ok := input.(string)
		if !ok {
			return nil, fmt.Errorf("parse_json input must be a string")
		}
		if strings.Contains(text, "timeout") {
			return map[string]any{"timeout": "OK"}, nil
		}
		return map[string]any{"name": "Veta"}, nil
	}))
	require.NoError(t, err)

	got, err := renderer.Render("page", nil)
	require.NoError(t, err)
	require.Equal(t, "Veta Raw Veta OK", got)
}

type testSafeHTML string

func (html testSafeHTML) SafeHTML() string {
	return string(html)
}

func TestRendererStructuralSafeHTML(t *testing.T) {
	files := fstest.MapFS{
		"page.j2": {Data: []byte(`{{ content }}`)},
	}
	renderer, err := New(files)
	require.NoError(t, err)

	got, err := renderer.Render("page", Context{"content": testSafeHTML("<strong>safe</strong>")})
	require.NoError(t, err)
	require.Equal(t, "<strong>safe</strong>", got)
}

func TestRendererAutoescapesByDefault(t *testing.T) {
	files := fstest.MapFS{
		"page.j2": {Data: []byte(`{{ page.content }}`)},
	}
	renderer, err := New(files)
	require.NoError(t, err)

	got, err := renderer.Render("page", Context{
		"page": map[string]any{"content": "<strong>escaped</strong>"},
	})
	require.NoError(t, err)
	require.Equal(t, "&lt;strong&gt;escaped&lt;/strong&gt;", got)
}

func TestRendererErrors(t *testing.T) {
	tests := []struct {
		name         string
		render       string
		wantErr      error
		wantContains []string
	}{
		{
			name:    "missing template",
			render:  "missing",
			wantErr: ErrTemplateNotFound,
		},
		{
			name:    "ambiguous template",
			render:  "ambiguous",
			wantErr: ErrTemplateAmbiguous,
			wantContains: []string{
				"ambiguous.html",
				"ambiguous.twig",
			},
		},
		{
			name:    "rejects parent traversal",
			render:  "../secret",
			wantErr: ErrTemplateNameInvalid,
		},
		{
			name:    "rejects absolute path",
			render:  "/page",
			wantErr: ErrTemplateNameInvalid,
		},
	}

	files := fstest.MapFS{
		"ambiguous.html": {Data: []byte("html")},
		"ambiguous.twig": {Data: []byte("twig")},
	}
	renderer, err := New(files)
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := renderer.Render(test.render, nil)
			require.Error(t, err)
			require.True(t, errors.Is(err, test.wantErr), "expected %v, got %v", test.wantErr, err)
			for _, expected := range test.wantContains {
				require.Contains(t, err.Error(), expected)
			}
		})
	}
}

func TestRendererIgnoresHiddenAndTemporaryTemplates(t *testing.T) {
	files := fstest.MapFS{
		".hidden.html":   {Data: []byte("hidden")},
		"base.html.tmp":  {Data: []byte("tmp")},
		"base.tmp":       {Data: []byte("tmp")},
		"base.twig~":     {Data: []byte("backup")},
		"partials/.keep": {Data: []byte("keep")},
		"partials/card":  {Data: []byte("card")},
		"partials/card~": {Data: []byte("backup")},
	}
	renderer, err := New(files)
	require.NoError(t, err)

	_, err = renderer.Render("base", nil)
	require.ErrorIs(t, err, ErrTemplateNotFound)

	_, err = renderer.Render("base.tmp", nil)
	require.ErrorIs(t, err, ErrTemplateNotFound)

	_, err = renderer.Render(".hidden.html", nil)
	require.ErrorIs(t, err, ErrTemplateNotFound)

	got, err := renderer.Render("partials/card", nil)
	require.NoError(t, err)
	require.Equal(t, "card", got)
}

func TestRendererContextErrors(t *testing.T) {
	files := fstest.MapFS{"page.j2": {Data: []byte("ok")}}
	renderer, err := New(files)
	require.NoError(t, err)

	_, err = renderer.Render("page", []string{"bad"})
	require.ErrorIs(t, err, ErrContextUnsupported)
}

func TestRendererOptionErrors(t *testing.T) {
	files := fstest.MapFS{"page.j2": {Data: []byte("ok")}}

	_, err := New(nil)
	require.ErrorIs(t, err, ErrTemplateFSRequired)

	_, err = New(files, WithFilter("", func(input, parameter any) (any, error) {
		return input, nil
	}))
	require.ErrorIs(t, err, ErrFilterNameInvalid)

	_, err = New(files, WithFilter("broken", nil))
	require.ErrorIs(t, err, ErrFilterNameInvalid)

	_, err = New(files, WithGlobal("", "bad"))
	require.ErrorIs(t, err, ErrGlobalNameInvalid)

	_, err = New(files, WithGlobal("broken", nil))
	require.ErrorIs(t, err, ErrGlobalNameInvalid)
}

func TestRendererWithExtensionsCompatibilityNoop(t *testing.T) {
	files := fstest.MapFS{
		"page.tpl": {Data: []byte("Hello {{ name }}")},
		"feed.xml": {Data: []byte("<title>{{ title }}</title>")},
	}
	renderer, err := New(files, WithExtensions("tpl"))
	require.NoError(t, err)

	got, err := renderer.Render("page", map[string]string{"name": "Veta"})
	require.NoError(t, err)
	require.Equal(t, "Hello Veta", got)

	got, err = renderer.Render("feed", map[string]string{"title": "Veta"})
	require.NoError(t, err)
	require.Equal(t, "<title>Veta</title>", got)
}
