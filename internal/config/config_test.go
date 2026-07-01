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
build:
  output: " public-build "
  clean: true
theme:
  source: " varavelio/veta-theme-clean@v1.0.0 "
dev:
  host: " 0.0.0.0 "
  port: 4000
  watch:
    - " content "
    - docs
html:
  minify: true
tailwindcss:
  stylesheets:
    - " css/app.css "
    - admin.css
  minify: true
`)},
		}

		config, err := Load(files)
		require.NoError(t, err)
		require.Equal(t, "public-build", config.Build.Output)
		require.True(t, config.Build.Clean)
		require.Equal(t, "0.0.0.0", config.Dev.Host)
		require.Equal(t, 4000, config.Dev.Port)
		require.Equal(t, []string{"content", "docs"}, config.Dev.Watch)
		require.True(t, config.HTML.Minify)
		require.Equal(t, "varavelio/veta-theme-clean@v1.0.0", config.Theme.Source)
		require.Equal(t, []string{"css/app.css", "admin.css"}, config.TailwindCSS.Stylesheets)
		require.True(t, config.TailwindCSS.Minify)
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
			want: Config{
				Build: Build{Output: DefaultBuildOutput},
				Dev:   Dev{Host: DefaultDevHost, Port: DefaultDevPort},
				Theme: Theme{Source: "./themes/basic"},
			},
		},
		{
			name: "build only",
			content: `
build:
  output: public
  clean: true
`,
			want: Config{
				Build: Build{Output: "public", Clean: true},
				Dev:   Dev{Host: DefaultDevHost, Port: DefaultDevPort},
			},
		},
		{
			name: "dev only",
			content: `
dev:
  host: 0.0.0.0
  port: 4000
  watch:
    - content
    - docs/reference
`,
			want: Config{
				Build: Build{Output: DefaultBuildOutput},
				Dev: Dev{
					Host:  "0.0.0.0",
					Port:  4000,
					Watch: []string{"content", "docs/reference"},
				},
			},
		},
		{
			name: "tailwind only",
			content: `
tailwindcss:
  stylesheets:
    - css/app.css
`,
			want: Config{
				Build:       Build{Output: DefaultBuildOutput},
				Dev:         Dev{Host: DefaultDevHost, Port: DefaultDevPort},
				TailwindCSS: TailwindCSS{Stylesheets: []string{"css/app.css"}},
			},
		},
		{
			name: "html minify",
			content: `
html:
  minify: true
`,
			want: Config{
				Build: Build{Output: DefaultBuildOutput},
				Dev:   Dev{Host: DefaultDevHost, Port: DefaultDevPort},
				HTML:  HTML{Minify: true},
			},
		},
		{
			name: "tailwind minify without stylesheets",
			content: `
tailwindcss:
  minify: true
`,
			want: Config{
				Build:       Build{Output: DefaultBuildOutput},
				Dev:         Dev{Host: DefaultDevHost, Port: DefaultDevPort},
				TailwindCSS: TailwindCSS{Minify: true},
			},
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
			name: "unknown build field",
			content: `
build:
  destination: dist
`,
		},
		{
			name: "unknown theme field",
			content: `
theme:
  source: ./theme
  name: clean
`,
		},
		{
			name: "unknown dev field",
			content: `
dev:
  hostname: localhost
`,
		},
		{
			name: "unknown html field",
			content: `
html:
  enabled: true
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
  stylesheets:
    - app.css
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

func TestDevValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr error
	}{
		{
			name: "absolute watch path",
			content: `
dev:
  watch:
    - /content
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "parent traversal watch path",
			content: `
dev:
  watch:
    - ../content
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "empty watch path",
			content: `
dev:
  watch:
    - ""
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "windows volume watch path",
			content: `
dev:
  watch:
    - C:\content
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "negative port",
			content: `
dev:
  port: -1
`,
			wantErr: ErrInvalid,
		},
		{
			name: "too large port",
			content: `
dev:
  port: 65536
`,
			wantErr: ErrInvalid,
		},
		{
			name:    "nul host",
			content: "dev:\n  host: \"bad\x00host\"\n",
			wantErr: ErrInvalid,
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

func TestTailwindValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr error
	}{
		{
			name: "absolute stylesheet path",
			content: `
tailwindcss:
  stylesheets:
    - /app.css
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "parent traversal stylesheet path",
			content: `
tailwindcss:
  stylesheets:
    - ../app.css
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "windows volume path",
			content: `
tailwindcss:
  stylesheets:
    - C:\public\app.css
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "empty stylesheet path",
			content: `
tailwindcss:
  stylesheets:
    - ""
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "duplicate stylesheet path",
			content: `
tailwindcss:
  stylesheets:
    - styles.css
    - " styles.css "
`,
			wantErr: ErrInvalid,
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

func TestBuildValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr error
	}{
		{
			name: "absolute output",
			content: `
build:
  output: /dist
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "parent traversal output",
			content: `
build:
  output: ../dist
`,
			wantErr: ErrPathInvalid,
		},
		{
			name: "windows volume output",
			content: `
build:
  output: C:\dist
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
