package tmpl

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
			name:   "renders pongo template",
			render: "pages/home",
			context: Context{
				"page": map[string]any{"title": "Home"},
			},
			want: "<h1>Home</h1>",
		},
		{
			name:   "renders html template fallback",
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
		"emails/welcome.html": {Data: []byte("Welcome {{ name }}")},
		"pages/home.pongo":    {Data: []byte("<h1>{{ page.title }}</h1>")},
		"plain.txt":           {Data: []byte("plain text")},
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
		"layouts/base.pongo": {Data: []byte(strings.Join([]string{
			"<html>",
			"<head><title>{{ page.title }}</title></head>",
			"<body>",
			"{% block content %}{% endblock %}",
			"{% include \"partials/footer\" %}",
			"</body>",
			"</html>",
		}, ""))},
		"pages/home.pongo": {Data: []byte(strings.Join([]string{
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
		"page.pongo": {Data: []byte(`{{ page.title|surround:"!" }} {{ page.content|trusted }}`)},
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

type testSafeHTML string

func (html testSafeHTML) SafeHTML() string {
	return string(html)
}

func TestRendererStructuralSafeHTML(t *testing.T) {
	files := fstest.MapFS{
		"page.pongo": {Data: []byte(`{{ content }}`)},
	}
	renderer, err := New(files)
	require.NoError(t, err)

	got, err := renderer.Render("page", Context{"content": testSafeHTML("<strong>safe</strong>")})
	require.NoError(t, err)
	require.Equal(t, "<strong>safe</strong>", got)
}

func TestRendererAutoescapesByDefault(t *testing.T) {
	files := fstest.MapFS{
		"page.pongo": {Data: []byte(`{{ page.content }}`)},
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
		name    string
		render  string
		wantErr error
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
		"ambiguous.html":  {Data: []byte("html")},
		"ambiguous.pongo": {Data: []byte("pongo")},
	}
	renderer, err := New(files)
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := renderer.Render(test.render, nil)
			require.Error(t, err)
			require.True(t, errors.Is(err, test.wantErr), "expected %v, got %v", test.wantErr, err)
		})
	}
}

func TestRendererContextErrors(t *testing.T) {
	files := fstest.MapFS{"page.pongo": {Data: []byte("ok")}}
	renderer, err := New(files)
	require.NoError(t, err)

	_, err = renderer.Render("page", []string{"bad"})
	require.ErrorIs(t, err, ErrContextUnsupported)
}

func TestRendererOptionErrors(t *testing.T) {
	files := fstest.MapFS{"page.pongo": {Data: []byte("ok")}}

	_, err := New(nil)
	require.ErrorIs(t, err, ErrTemplateFSRequired)

	_, err = New(files, WithExtensions())
	require.ErrorIs(t, err, ErrTemplateNameInvalid)

	_, err = New(files, WithFilter("", func(input, parameter any) (any, error) {
		return input, nil
	}))
	require.ErrorIs(t, err, ErrFilterNameInvalid)

	_, err = New(files, WithFilter("broken", nil))
	require.ErrorIs(t, err, ErrFilterNameInvalid)
}

func TestRendererCustomExtensions(t *testing.T) {
	files := fstest.MapFS{
		"page.tpl": {Data: []byte("Hello {{ name }}")},
		"feed.xml": {Data: []byte("<title>{{ title }}</title>")},
	}
	renderer, err := New(files, WithExtensions("tpl"))
	require.NoError(t, err)

	got, err := renderer.Render("page", map[string]string{"name": "Veta"})
	require.NoError(t, err)
	require.Equal(t, "Hello Veta", got)

	renderer, err = New(files, WithExtensions("xml"))
	require.NoError(t, err)

	got, err = renderer.Render("feed", map[string]string{"title": "Veta"})
	require.NoError(t, err)
	require.Equal(t, "<title>Veta</title>", got)
}
