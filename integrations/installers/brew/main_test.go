package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseManifest verifies Homebrew manifest parsing.
func TestParseManifest(t *testing.T) {
	t.Run("returns archive hashes by filename", func(t *testing.T) {
		checksums, err := parseManifest([]byte(`{
  "artifacts": [
    { "name": "veta_linux_amd64.tar.gz", "sha256": "abc" },
    { "name": "veta_darwin_arm64.tar.gz", "sha256": "def" }
  ]
}`))
		require.NoError(t, err)
		require.Equal(t, "abc", checksums["veta_linux_amd64.tar.gz"])
		require.Equal(t, "def", checksums["veta_darwin_arm64.tar.gz"])
	})

	t.Run("rejects malformed manifest content", func(t *testing.T) {
		_, err := parseManifest([]byte(`not-json`))
		require.Error(t, err)
	})

	t.Run("rejects incomplete artifacts", func(t *testing.T) {
		_, err := parseManifest([]byte(`{ "artifacts": [{ "name": "veta_linux_amd64.tar.gz" }] }`))
		require.Error(t, err)
	})
}

// TestWriteFormula verifies Homebrew formula file placement.
func TestWriteFormula(t *testing.T) {
	t.Run("writes formula into the Formula/veta directory", func(t *testing.T) {
		outputRoot := t.TempDir()
		err := writeFormula(outputRoot, "veta.rb", "0.1.0", map[string]string{
			"veta_darwin_arm64.tar.gz": "darwin-arm64",
			"veta_darwin_amd64.tar.gz": "darwin-amd64",
			"veta_linux_arm64.tar.gz":  "linux-arm64",
			"veta_linux_amd64.tar.gz":  "linux-amd64",
		})
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(outputRoot, "Formula", "veta", "veta.rb"))
		require.NoError(t, err)
		require.Contains(t, string(content), "class Veta < Formula")
	})
}

// TestFormulaClassName verifies formula filename to Ruby class conversion.
func TestFormulaClassName(t *testing.T) {
	cases := map[string]string{
		"veta.rb":       "Veta",
		"veta-next.rb":  "VetaNext",
		"veta@0.1.0.rb": "VetaAT010",
	}
	for fileName, expected := range cases {
		t.Run(fileName, func(t *testing.T) {
			require.Equal(t, expected, formulaClassName(fileName))
		})
	}
}

// TestGenerateFormula verifies formula generation with required checksums.
func TestGenerateFormula(t *testing.T) {
	t.Run("generates platform selectors", func(t *testing.T) {
		formula, err := generateFormula("veta.rb", "0.1.0", map[string]string{
			"veta_darwin_arm64.tar.gz": "darwin-arm64",
			"veta_darwin_amd64.tar.gz": "darwin-amd64",
			"veta_linux_arm64.tar.gz":  "linux-arm64",
			"veta_linux_amd64.tar.gz":  "linux-amd64",
		})
		require.NoError(t, err)
		require.Contains(t, formula, "class Veta < Formula")
		require.Contains(t, formula, "veta_linux_amd64.tar.gz")
		require.Contains(t, formula, "sha256 \"linux-amd64\"")
	})

	t.Run("requires all release archive checksums", func(t *testing.T) {
		_, err := generateFormula("veta.rb", "0.1.0", map[string]string{
			"veta_darwin_arm64.tar.gz": "darwin-arm64",
		})
		require.Error(t, err)
	})
}
