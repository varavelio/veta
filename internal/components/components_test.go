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
		"components/ui/button.pongo": {Data: []byte("button")},
		"components/ui/card.pongo":   {Data: []byte("card")},
	}, renderer, WithSlotRenderer(func(content string, _ any) (string, error) {
		return "slot(" + content + ")", nil
	}))
	require.NoError(t, err)

	got, err := processor.Render(
		`<ui-card title="Sale">Hello <ui-button text="Buy" /></ui-card>`,
		map[string]any{
			"data": map[string]any{"site": map[string]any{"name": "Veta"}},
			"page": map[string]any{"title": "Home"},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		`<components/ui/card>slot(Hello <components/ui/button>Buyslot()</components/ui/button>)</components/ui/card>`,
		got,
	)
	require.Len(t, renderer.calls, 2)
	require.Equal(t, "components/ui/button", renderer.calls[0].template)
	require.Equal(t, "components/ui/card", renderer.calls[1].template)
	require.Equal(
		t,
		map[string]any{
			"content": SafeHTML(
				"slot(Hello <components/ui/button>Buyslot()</components/ui/button>)",
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
				Template: "components/ui-table",
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

	_, err = New(fstest.MapFS{"components/readme.md": {Data: []byte("bad")}}, &recordingRenderer{})
	require.ErrorIs(t, err, ErrFormatUnsupported)

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
