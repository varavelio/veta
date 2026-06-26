package config

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("loads default file", func(t *testing.T) {
		files := fstest.MapFS{
			FileName: {Data: []byte(`
theme:
  source: " varavelio/veta-theme-clean@v1.0.0 "
tailwindcss:
  input: " public/css/app.css "
  output: " dist/css/app.css "
  minify: true
`)},
		}

		config, err := Load(files)
		require.NoError(t, err)
		require.Equal(t, "varavelio/veta-theme-clean@v1.0.0", config.Theme.Source)
		require.Equal(t, "public/css/app.css", config.TailwindCSS.Input)
		require.Equal(t, "dist/css/app.css", config.TailwindCSS.Output)
		require.Equal(t, true, config.TailwindCSS.Minify)
		require.True(t, config.Theme.Enabled())
		require.True(t, config.TailwindCSS.Enabled())
	})

	t.Run("missing file returns defaults", func(t *testing.T) {
		config, err := Load(fstest.MapFS{})
		require.NoError(t, err)
		require.Equal(t, Default(), config)
		require.False(t, config.Theme.Enabled())
		require.False(t, config.TailwindCSS.Enabled())
	})

	t.Run("loads alternate file names", func(t *testing.T) {
		for _, fileName := range []string{"veta.yml", ".veta.yaml", ".veta.yml"} {
			t.Run(fileName, func(t *testing.T) {
				config, err := Load(fstest.MapFS{
					fileName: {Data: []byte(`theme: { source: "./theme" }`)},
				})
				require.NoError(t, err)
				require.Equal(t, "./theme", config.Theme.Source)
			})
		}
	})

	t.Run("uses documented priority order", func(t *testing.T) {
		config, err := Load(fstest.MapFS{
			".veta.yml":  {Data: []byte(`theme: { source: "./dot-yml" }`)},
			".veta.yaml": {Data: []byte(`theme: { source: "./dot-yaml" }`)},
			"veta.yml":   {Data: []byte(`theme: { source: "./yml" }`)},
			"veta.yaml":  {Data: []byte(`theme: { source: "./yaml" }`)},
		})
		require.NoError(t, err)
		require.Equal(t, "./yaml", config.Theme.Source)
	})

	t.Run("invalid higher priority file stops loading", func(t *testing.T) {
		_, err := Load(fstest.MapFS{
			"veta.yaml": {Data: []byte(`site: { title: Veta }`)},
			"veta.yml":  {Data: []byte(`theme: { source: "./theme" }`)},
		})
		require.ErrorIs(t, err, ErrInvalid)
	})

	t.Run("custom file", func(t *testing.T) {
		files := fstest.MapFS{
			"nested/custom.yaml": {Data: []byte(`theme: { source: "./theme" }`)},
		}

		config, err := LoadFile(files, "nested/custom.yaml")
		require.NoError(t, err)
		require.Equal(t, "./theme", config.Theme.Source)
	})
}

func TestLoadErrors(t *testing.T) {
	_, err := Load(nil)
	require.ErrorIs(t, err, ErrFSRequired)

	_, err = LoadFile(fstest.MapFS{}, "../veta.yaml")
	require.ErrorIs(t, err, ErrPathInvalid)

	_, err = Load(fstest.MapFS{
		FileName: {Data: []byte(`tailwindcss: [`)},
	})
	require.ErrorIs(t, err, ErrInvalid)
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    Config
	}{
		{
			name:    "empty",
			content: "",
			want:    Default(),
		},
		{
			name:    "whitespace",
			content: "\n\t  ",
			want:    Default(),
		},
		{
			name: "theme only",
			content: `
theme:
  source: ./themes/basic
`,
			want: Config{Theme: Theme{Source: "./themes/basic"}},
		},
		{
			name: "tailwind only",
			content: `
tailwindcss:
  input: public/css/app.css
  output: dist/css/app.css
`,
			want: Config{TailwindCSS: TailwindCSS{Input: "public/css/app.css", Output: "dist/css/app.css"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := Parse([]byte(test.content))
			require.NoError(t, err)
			require.Equal(t, test.want, config)
		})
	}
}

func TestParseStrictSchema(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "unknown top-level field",
			content: `
site:
  title: Veta
`,
		},
		{
			name: "unknown theme field",
			content: `
theme:
  sha256: abc
`,
		},
		{
			name: "unknown tailwind field",
			content: `
tailwindcss:
  enabled: true
`,
		},
		{
			name: "multiple documents",
			content: `
theme:
  source: ./theme
---
tailwindcss:
  input: public/app.css
  output: dist/app.css
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.content))
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func TestTailwindValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr error
	}{
		{
			name: "input without output",
			content: `
tailwindcss:
  input: public/app.css
`,
			wantErr: ErrInvalid,
		},
		{
			name: "output without input",
			content: `
tailwindcss:
  output: dist/app.css
`,
			wantErr: ErrInvalid,
		},
		{
			name: "minify without input output",
			content: `
tailwindcss:
  minify: true
`,
			wantErr: ErrInvalid,
		},
		{
			name: "absolute input",
			content: `
tailwindcss:
  input: /public/app.css
  output: dist/app.css
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "parent traversal output",
			content: `
tailwindcss:
  input: public/app.css
  output: ../dist/app.css
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "windows volume path",
			content: `
tailwindcss:
  input: C:\public\app.css
  output: dist/app.css
`,
			wantErr: ErrPathInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.content))
			require.Error(t, err)
			require.True(t, errors.Is(err, test.wantErr), "expected %v, got %v", test.wantErr, err)
		})
	}
}

func TestThemeValidation(t *testing.T) {
	_, err := Parse([]byte("theme:\n  source: \"bad\x00source\"\n"))
	require.ErrorIs(t, err, ErrInvalid)
}

func TestCleanConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "clean relative", path: "./veta.yaml", want: "veta.yaml"},
		{name: "nested", path: "config/veta.yaml", want: "config/veta.yaml"},
		{name: "backslash", path: `config\veta.yaml`, want: "config/veta.yaml"},
		{name: "empty", path: "", wantErr: true},
		{name: "root", path: ".", wantErr: true},
		{name: "absolute", path: "/veta.yaml", wantErr: true},
		{name: "parent", path: "../veta.yaml", wantErr: true},
		{name: "windows volume", path: `C:\veta.yaml`, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := cleanConfigPath(test.path)
			if test.wantErr {
				require.ErrorIs(t, err, ErrPathInvalid)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestLoadFileWrapsUnexpectedReadErrors(t *testing.T) {
	_, err := LoadFile(failingFS{}, FileName)
	require.Error(t, err)
	require.False(t, errors.Is(err, fs.ErrNotExist))
}

type failingFS struct{}

func (failingFS) Open(string) (fs.File, error) {
	return nil, errors.New("boom")
}
