package vfs

import (
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"
)

func BenchmarkOverlayRepeatedStatAndReadDir(b *testing.B) {
	theme := fstest.MapFS{}
	project := fstest.MapFS{}
	for index := range 250 {
		theme[fmt.Sprintf("templates/theme-%03d.j2", index)] = &fstest.MapFile{
			Data: []byte("theme"),
		}
		project[fmt.Sprintf("templates/project-%03d.j2", index)] = &fstest.MapFile{
			Data: []byte("project"),
		}
	}
	project["templates/base.j2"] = &fstest.MapFile{Data: []byte("base")}

	overlay, err := NewOverlay(
		Layer{Name: "theme", FS: theme},
		Layer{Name: "project", FS: project},
	)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := fs.Stat(overlay, "templates/base.j2"); err != nil {
			b.Fatal(err)
		}
		if _, err := fs.ReadDir(overlay, "templates"); err != nil {
			b.Fatal(err)
		}
	}
}
