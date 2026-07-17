package components

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

type recordingRenderer struct {
	outputs map[string]string
	calls   []renderCall
}

type renderCall struct {
	context  map[string]any
	template string
}

type templateRendererFunc func(name string, context any) (string, error)

// Render calls the underlying test renderer function.
func (renderer templateRendererFunc) Render(name string, context any) (string, error) {
	return renderer(name, context)
}

func (renderer *recordingRenderer) Render(name string, context any) (string, error) {
	contextMap, _ := context.(map[string]any)
	renderer.calls = append(renderer.calls, renderCall{context: contextMap, template: name})
	if output, ok := renderer.outputs[name]; ok {
		return output, nil
	}

	props, _ := contextMap["props"].(map[string]any)
	content := fmt.Sprint(props["content"])
	return fmt.Sprintf("<%s>%s%s</%s>", name, propString(props, "text"), content, name), nil
}

// propString returns a string prop from a test renderer context.
func propString(props map[string]any, key string) string {
	value, _ := props[key].(string)
	return value
}

// TestProcessorRender verifies self-closing, paired, nested components and slot
// rendering.
func TestProcessorRender(t *testing.T) {
	renderer := &recordingRenderer{}
	processor, err := New(fstest.MapFS{
		"components/ui/button.twig": {Data: []byte("button")},
		"components/ui/card.html":   {Data: []byte("card")},
	}, renderer, WithSlotRenderer(func(content string, _ any) (string, error) {
		return "slot(" + content + ")", nil
	}))
	require.NoError(t, err)

	got, err := processor.Render(
		`<ui-card title="Sale">Hello <ui-button text="Buy" /></ui-card>`,
		// The base context mirrors the root template context passed by render.
		map[string]any{
			"data":  map[string]any{"site": map[string]any{"name": "Veta"}},
			"pages": []map[string]any{{"title": "Home", "permalink": "/"}},
			"page":  map[string]any{"title": "Home"},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		`<components/ui/card.html>slot(Hello <components/ui/button.twig>Buyslot()</components/ui/button.twig>)</components/ui/card.html>`,
		got,
	)
	require.Len(t, renderer.calls, 2)
	require.Equal(t, "components/ui/button.twig", renderer.calls[0].template)
	require.Equal(t, "components/ui/card.html", renderer.calls[1].template)
	require.Equal(
		t,
		map[string]any{
			"content": SafeHTML(
				"slot(Hello <components/ui/button.twig>Buyslot()</components/ui/button.twig>)",
			),
			"title": "Sale",
		},
		renderer.calls[1].context["props"],
	)
	require.Equal(
		t,
		map[string]any{"site": map[string]any{"name": "Veta"}},
		renderer.calls[1].context["data"],
	)
	require.Equal(
		t,
		[]map[string]any{{"title": "Home", "permalink": "/"}},
		renderer.calls[1].context["pages"],
	)
}

// TestProcessorRenderMultilineTags verifies paired and self-closing component
// invocations accept HTML-like attributes spread across lines.
func TestProcessorRenderMultilineTags(t *testing.T) {
	renderer := &recordingRenderer{outputs: map[string]string{
		"components/card.j2": "<article>rendered</article>",
	}}
	processor, err := New(
		fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
		renderer,
	)
	require.NoError(t, err)
	content := strings.Join([]string{
		"<card",
		`title="Paired"`,
		`description="A > B"`,
		">slot</card>",
		"<card",
		`title="Self Closing"`,
		"/>",
	}, "\n")

	got, err := processor.Render(content, nil)
	require.NoError(t, err)
	require.Equal(t, "<article>rendered</article>\n<article>rendered</article>", got)
	require.Len(t, renderer.calls, 2)
	require.Equal(t, "Paired", renderer.calls[0].context["props"].(map[string]any)["title"])
	require.Equal(t, "A > B", renderer.calls[0].context["props"].(map[string]any)["description"])
	require.Equal(
		t,
		SafeHTML("slot"),
		renderer.calls[0].context["props"].(map[string]any)["content"],
	)
	require.Equal(t, "Self Closing", renderer.calls[1].context["props"].(map[string]any)["title"])
	require.Equal(t, SafeHTML(""), renderer.calls[1].context["props"].(map[string]any)["content"])
}

// TestProcessorRenderIgnoresUnregisteredAndProtectedTags verifies that regular
// HTML and code examples are left untouched.
func TestProcessorRenderIgnoresUnregisteredAndProtectedTags(t *testing.T) {
	processor, err := New(fstest.MapFS{
		"components/card.j2": {Data: []byte("card")},
	}, &recordingRenderer{})
	require.NoError(t, err)

	content := "<div>HTML</div> `inline <card />`\n\n```html\n<card />\n```"
	got, err := processor.Render(content, nil)
	require.NoError(t, err)
	require.Equal(t, content, got)
}

// TestProcessorRenderArbitraryContent verifies that component expansion remains
// lexical and preserves unrelated content across common input formats.
func TestProcessorRenderArbitraryContent(t *testing.T) {
	processor, err := New(
		fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
		&recordingRenderer{outputs: map[string]string{
			"components/card.j2": "<article>rendered</article>",
		}},
	)
	require.NoError(t, err)

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "plain text",
			content: "Use any text, including 2 < 3 and symbols {}[]().",
			want:    "Use any text, including 2 < 3 and symbols {}[]().",
		},
		{
			name:    "plain text component",
			content: "before <card /> after",
			want:    "before <article>rendered</article> after",
		},
		{
			name:    "markdown",
			content: "# Heading\n\nText with **formatting**.\n\n<card />",
			want:    "# Heading\n\nText with **formatting**.\n\n<article>rendered</article>",
		},
		{
			name:    "json string",
			content: `{"type":"example","content":"<card />","native":"<span>ok</span>"}`,
			want:    `{"type":"example","content":"<article>rendered</article>","native":"<span>ok</span>"}`,
		},
		{
			name:    "complex html",
			content: `<main><section data-kind="native"><card /></section><custom-element>unchanged</custom-element></main>`,
			want:    `<main><section data-kind="native"><article>rendered</article></section><custom-element>unchanged</custom-element></main>`,
		},
		{
			name:    "malformed unknown tag",
			content: `<unknown title="<card />"`,
			want:    `<unknown title="<card />"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := processor.Render(test.content, nil)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

// TestProcessorRenderPreservesNonTagHTMLContexts verifies that component-like
// text in HTML metadata and raw-text elements remains byte-for-byte unchanged.
func TestProcessorRenderPreservesNonTagHTMLContexts(t *testing.T) {
	processor, err := New(
		fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
		&recordingRenderer{outputs: map[string]string{
			"components/card.j2": "<article>rendered</article>",
		}},
	)
	require.NoError(t, err)

	content := strings.Join([]string{
		`<!-- <card /> -->`,
		`<div data-example="<card />"><card /></div>`,
		`<div data-example="<script>"><card /></div>`,
		`<div data-example="<pre title='>'>"><card /></div>`,
		"<div data-example=\"`\"><card /></div> `code`",
		"<card title=\"`\">slot</card> `code`",
		`<DIV data-example="<card />"><card /></DIV>`,
		`<script>const sample = "<card />";</script>`,
		`<style>.sample::before { content: "<card />"; }</style>`,
		`<code><card /></code>`,
		`<pre><card /></pre>`,
		`<textarea><card /></textarea>`,
		`<title><card /></title>`,
	}, "\n")
	want := strings.Replace(
		content,
		`<div data-example="<card />"><card /></div>`,
		`<div data-example="<card />"><article>rendered</article></div>`,
		1,
	)
	want = strings.Replace(
		want,
		`<div data-example="<script>"><card /></div>`,
		`<div data-example="<script>"><article>rendered</article></div>`,
		1,
	)
	want = strings.Replace(
		want,
		`<div data-example="<pre title='>'>"><card /></div>`,
		`<div data-example="<pre title='>'>"><article>rendered</article></div>`,
		1,
	)
	want = strings.Replace(
		want,
		"<div data-example=\"`\"><card /></div> `code`",
		"<div data-example=\"`\"><article>rendered</article></div> `code`",
		1,
	)
	want = strings.Replace(
		want,
		"<card title=\"`\">slot</card> `code`",
		"<article>rendered</article> `code`",
		1,
	)
	want = strings.Replace(
		want,
		`<DIV data-example="<card />"><card /></DIV>`,
		`<DIV data-example="<card />"><article>rendered</article></DIV>`,
		1,
	)

	got, err := processor.Render(content, nil)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// TestProcessorRenderRequiresExactRegisteredTagNames verifies that names which
// merely start with a registered component name remain unchanged.
func TestProcessorRenderRequiresExactRegisteredTagNames(t *testing.T) {
	processor, err := New(
		fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
		&recordingRenderer{},
	)
	require.NoError(t, err)

	content := strings.Join([]string{
		`<cardinal>text</cardinal>`,
		`<card:part data-example="<card />">text</card:part>`,
		`<card.component>text</card.component>`,
		`<card_name>text</card_name>`,
		`<card@part>text</card@part>`,
	}, "\n")

	got, err := processor.Render(content, nil)
	require.NoError(t, err)
	require.Equal(t, content, got)
}

// TestProcessorRenderProtectsMarkdownCode verifies valid inline and fenced code
// delimiters prevent component expansion.
func TestProcessorRenderProtectsMarkdownCode(t *testing.T) {
	processor, err := New(
		fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
		&recordingRenderer{outputs: map[string]string{
			"components/card.j2": "rendered",
		}},
	)
	require.NoError(t, err)

	content := strings.Join([]string{
		"`<card />`",
		"`<script>`",
		"``<card />``",
		"````html",
		"<card />",
		"<script>",
		"```",
		"<card />",
		"````",
		"<card />",
	}, "\n")
	want := strings.TrimSuffix(content, "<card />") + "rendered"

	got, err := processor.Render(content, nil)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// TestProcessorRenderDoesNotRescanOutput verifies component output is a final
// one-pass replacement rather than a new component source.
func TestProcessorRenderDoesNotRescanOutput(t *testing.T) {
	renderer := &recordingRenderer{outputs: map[string]string{
		"components/card.j2": "<card />",
	}}
	processor, err := New(
		fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
		renderer,
	)
	require.NoError(t, err)

	got, err := processor.Render("<card />", nil)
	require.NoError(t, err)
	require.Equal(t, "<card />", got)
	require.Len(t, renderer.calls, 1)
}

// TestProcessorRenderLimitsRecursion verifies recursive re-entry and deeply
// nested source fail deterministically instead of exhausting the stack.
func TestProcessorRenderLimitsRecursion(t *testing.T) {
	t.Run("renderer re-entry", func(t *testing.T) {
		var processor *Processor
		renderer := templateRendererFunc(func(_ string, context any) (string, error) {
			return processor.Render("<card />", context)
		})
		var err error
		processor, err = New(
			fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
			renderer,
		)
		require.NoError(t, err)

		_, err = processor.Render("<card />", nil)
		require.ErrorIs(t, err, ErrRenderLimit)
	})

	t.Run("deep source nesting", func(t *testing.T) {
		processor, err := New(
			fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
			&recordingRenderer{},
		)
		require.NoError(t, err)
		content := strings.Repeat("<card>", maxRenderDepth+1) +
			strings.Repeat("</card>", maxRenderDepth+1)

		_, err = processor.Render(content, nil)
		require.ErrorIs(t, err, ErrRenderLimit)
	})
}

// TestProcessorArbitraryExtensions verifies component discovery does not filter
// by template extension.
func TestProcessorArbitraryExtensions(t *testing.T) {
	processor, err := New(fstest.MapFS{
		"components/note.html":      {Data: []byte("note")},
		"components/readme.md":      {Data: []byte("readme")},
		"components/ui/button.twig": {Data: []byte("button")},
	}, &recordingRenderer{})
	require.NoError(t, err)

	require.Equal(
		t,
		[]Component{
			{
				Depth:    0,
				Path:     "components/note.html",
				Tag:      "note",
				Template: "components/note.html",
			},
			{
				Depth:    0,
				Path:     "components/readme.md",
				Tag:      "readme",
				Template: "components/readme.md",
			},
			{
				Depth:    1,
				Path:     "components/ui/button.twig",
				Tag:      "ui-button",
				Template: "components/ui/button.twig",
			},
		},
		processor.Components(),
	)
}

// TestProcessorIgnoresHiddenAndTemporaryFiles verifies editor and dot files are
// not registered as components.
func TestProcessorIgnoresHiddenAndTemporaryFiles(t *testing.T) {
	processor, err := New(fstest.MapFS{
		"components/.DS_Store":     {Data: []byte("ignored")},
		"components/.gitkeep":      {Data: []byte("ignored")},
		"components/.hidden/card":  {Data: []byte("ignored")},
		"components/card.html.tmp": {Data: []byte("ignored")},
		"components/card.twig~":    {Data: []byte("ignored")},
		"components/note.html":     {Data: []byte("note")},
		"components/note.tmp":      {Data: []byte("ignored")},
	}, &recordingRenderer{})
	require.NoError(t, err)

	require.Equal(
		t,
		[]Component{
			{
				Depth:    0,
				Path:     "components/note.html",
				Tag:      "note",
				Template: "components/note.html",
			},
		},
		processor.Components(),
	)

	processor, err = New(fstest.MapFS{
		"components/.gitkeep": {Data: []byte("ignored")},
		"components/card.tmp": {Data: []byte("ignored")},
	}, nil)
	require.NoError(t, err)
	require.Nil(t, processor.Components())
}

// TestProcessorConflicts verifies top-down component conflict resolution.
func TestProcessorConflicts(t *testing.T) {
	processor, err := New(fstest.MapFS{
		"components/ui-table.j2": {Data: []byte("root")},
		"components/ui/table.j2": {Data: []byte("nested")},
	}, &recordingRenderer{})
	require.NoError(t, err)

	require.Equal(
		t,
		[]Component{
			{
				Depth:    0,
				Path:     "components/ui-table.j2",
				Tag:      "ui-table",
				Template: "components/ui-table.j2",
			},
		},
		processor.Components(),
	)
	require.Equal(
		t,
		[]Conflict{
			{
				Ignored: "components/ui/table.j2",
				Tag:     "ui-table",
				Winner:  "components/ui-table.j2",
			},
		},
		processor.Conflicts(),
	)
}

// TestProcessorConflictsSameStemDifferentExtensions verifies duplicate stems are
// resolved deterministically without making rendering ambiguous.
func TestProcessorConflictsSameStemDifferentExtensions(t *testing.T) {
	processor, err := New(fstest.MapFS{
		"components/ui/button":      {Data: []byte("extensionless")},
		"components/ui/button.html": {Data: []byte("html")},
		"components/ui/button.twig": {Data: []byte("twig")},
	}, &recordingRenderer{})
	require.NoError(t, err)

	require.Equal(
		t,
		[]Component{
			{
				Depth:    1,
				Path:     "components/ui/button.html",
				Tag:      "ui-button",
				Template: "components/ui/button.html",
			},
		},
		processor.Components(),
	)
	require.Equal(
		t,
		[]Conflict{
			{
				Ignored: "components/ui/button.twig",
				Tag:     "ui-button",
				Winner:  "components/ui/button.html",
			},
			{
				Ignored: "components/ui/button",
				Tag:     "ui-button",
				Winner:  "components/ui/button.html",
			},
		},
		processor.Conflicts(),
	)

	got, err := processor.Render("<ui-button />", nil)
	require.NoError(t, err)
	require.Equal(t, "<components/ui/button.html></components/ui/button.html>", got)
}

// TestNewMissingDirectory verifies that missing component directories are valid.
func TestNewMissingDirectory(t *testing.T) {
	processor, err := New(fstest.MapFS{}, nil)
	require.NoError(t, err)

	got, err := processor.Render("hello", nil)
	require.NoError(t, err)
	require.Equal(t, "hello", got)
}

// TestNewErrors verifies component discovery validation.
func TestNewErrors(t *testing.T) {
	_, err := New(nil, nil)
	require.ErrorIs(t, err, ErrFSRequired)

	_, err = New(fstest.MapFS{"components/card.j2": {Data: []byte("card")}}, nil)
	require.ErrorIs(t, err, ErrRendererRequired)

	_, err = New(fstest.MapFS{"components/Bad.j2": {Data: []byte("bad")}}, &recordingRenderer{})
	require.ErrorIs(t, err, ErrComponentNameInvalid)
}

// TestRenderErrors verifies component tag syntax validation.
func TestRenderErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr error
	}{
		{name: "missing close", content: `<card>`, wantErr: ErrSyntax},
		{name: "unexpected close", content: `</card>`, wantErr: ErrSyntax},
		{name: "malformed attribute", content: `<card title=test />`, wantErr: ErrAttributeInvalid},
	}

	processor, err := New(
		fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
		&recordingRenderer{},
	)
	require.NoError(t, err)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := processor.Render(test.content, nil)
			require.ErrorIs(t, err, test.wantErr)
		})
	}
}
