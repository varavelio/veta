package components

import (
	"strings"
	"testing"
	"testing/fstest"
)

func BenchmarkProcessorRenderProtectedRanges(b *testing.B) {
	processor, err := New(
		fstest.MapFS{"components/card.j2": {Data: []byte("card")}},
		&recordingRenderer{
			outputs: map[string]string{"components/card.j2": "<section>card</section>"},
		},
	)
	if err != nil {
		b.Fatal(err)
	}

	var content strings.Builder
	for range 100 {
		content.WriteString("`inline <card /> ignored`\n")
		content.WriteString("```html\n<card>ignored</card>\n```\n")
		content.WriteString("<card />\n")
	}
	input := content.String()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := processor.Render(input, nil); err != nil {
			b.Fatal(err)
		}
	}
}
