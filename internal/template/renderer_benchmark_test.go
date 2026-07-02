package template

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

func BenchmarkRendererRepeatedExtensionlessRender(b *testing.B) {
	files := fstest.MapFS{
		"layouts/base.j2": {Data: []byte("<html>{{ page.title }}</html>")},
	}
	renderer, err := New(files)
	if err != nil {
		b.Fatal(err)
	}
	context := Context{"page": map[string]any{"title": "Veta"}}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := renderer.Render("layouts/base", context); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRendererJSFilterLoop(b *testing.B) {
	var templateContent strings.Builder
	for index := range 100 {
		fmt.Fprintf(&templateContent, `{{ "%d"|identity }}`, index)
	}
	files := fstest.MapFS{"page.j2": {Data: []byte(templateContent.String())}}
	renderer, err := New(files, WithFilter("identity", func(input, _ any) (any, error) {
		return input, nil
	}))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := renderer.Render("page", nil); err != nil {
			b.Fatal(err)
		}
	}
}
