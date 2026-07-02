package output

import (
	"fmt"
	"testing"
)

func BenchmarkWriterWriteManyFiles(b *testing.B) {
	files := make([]File, 0, 1000)
	for index := range 1000 {
		files = append(files, File{
			Content: []byte("hello"),
			Path:    fmt.Sprintf("posts/%03d/index.html", index),
		})
	}

	b.ReportAllocs()
	for iteration := range b.N {
		writer, err := New(b.TempDir())
		if err != nil {
			b.Fatal(err)
		}
		if err := writer.Write(files); err != nil {
			b.Fatalf("iteration %d: %v", iteration, err)
		}
	}
}
