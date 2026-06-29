package components

import (
	"fmt"
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

// TestProcessorRenderIgnoresUnregisteredAndProtectedTags verifies that regular
// HTML and code examples are left untouched.
func TestProcessorRenderIgnoresUnregisteredAndProtectedTags(t *testing.T) {
	processor, err := New(fstest.MapFS{
		"components/card.pongo": {Data: []byte("card")},
	}, &recordingRenderer{})
	require.NoError(t, err)

	content := "<div>HTML</div> `inline <card />`\n\n```html\n<card />\n```"
	got, err := processor.Render(content, nil)
	require.NoError(t, err)
	require.Equal(t, content, got)
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
		"components/ui-table.pongo": {Data: []byte("root")},
		"components/ui/table.pongo": {Data: []byte("nested")},
	}, &recordingRenderer{})
	require.NoError(t, err)

	require.Equal(
		t,
		[]Component{
			{
				Depth:    0,
				Path:     "components/ui-table.pongo",
				Tag:      "ui-table",
				Template: "components/ui-table.pongo",
			},
		},
		processor.Components(),
	)
	require.Equal(
		t,
		[]Conflict{
			{
				Ignored: "components/ui/table.pongo",
				Tag:     "ui-table",
				Winner:  "components/ui-table.pongo",
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

	_, err = New(fstest.MapFS{"components/card.pongo": {Data: []byte("card")}}, nil)
	require.ErrorIs(t, err, ErrRendererRequired)

	_, err = New(fstest.MapFS{"components/Bad.pongo": {Data: []byte("bad")}}, &recordingRenderer{})
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
		fstest.MapFS{"components/card.pongo": {Data: []byte("card")}},
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
