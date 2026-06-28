package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArchiveName verifies release archive naming conventions.
func TestArchiveName(t *testing.T) {
	cases := map[string]struct {
		target   target
		expected string
	}{
		"darwin arm64": {
			target:   target{OS: "darwin", Arch: "arm64"},
			expected: "veta_darwin_arm64.tar.gz",
		},
		"linux amd64": {
			target:   target{OS: "linux", Arch: "amd64"},
			expected: "veta_linux_amd64.tar.gz",
		},
		"windows amd64": {
			target:   target{OS: "windows", Arch: "amd64"},
			expected: "veta_windows_amd64.zip",
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, testCase.expected, archiveName(testCase.target))
		})
	}
}

// TestBinaryName verifies the binary name used inside archives.
func TestBinaryName(t *testing.T) {
	cases := map[string]string{
		"linux":   "veta",
		"windows": "veta.exe",
	}
	for goos, expected := range cases {
		t.Run(goos, func(t *testing.T) {
			require.Equal(t, expected, binaryName(goos))
		})
	}
}

// TestWriteManifest verifies structured release manifest output.
func TestWriteManifest(t *testing.T) {
	t.Run("writes release metadata and artifacts", func(t *testing.T) {
		distDir := t.TempDir()
		metadata := releaseMetadata{
			Commit:  "abc123",
			Date:    "2026-01-02T03:04:05Z",
			Version: "0.1.0",
		}
		artifacts := []releaseArtifact{{
			Arch:   "amd64",
			Format: "tar.gz",
			Name:   "veta_linux_amd64.tar.gz",
			OS:     "linux",
			SHA256: "sha",
		}}

		err := writeManifest(distDir, metadata, artifacts)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(distDir, manifestFileName))
		require.NoError(t, err)

		var manifest releaseManifest
		require.NoError(t, json.Unmarshal(content, &manifest))
		require.Equal(t, "0.1.0", manifest.Version)
		require.Equal(t, "abc123", manifest.Commit)
		require.Equal(t, artifacts, manifest.Artifacts)
	})
}

// TestNormalizeVersion verifies release version normalization.
func TestNormalizeVersion(t *testing.T) {
	cases := map[string]string{
		" V0.1.0-BETA.1 ":  "0.1.0-beta.1",
		"refs/tags/v0.1.0": "0.1.0",
		"v0.1.0":           "0.1.0",
	}
	for value, expected := range cases {
		t.Run(value, func(t *testing.T) {
			require.Equal(t, expected, normalizeVersion(value))
		})
	}
}

// TestFirstNonEmpty verifies fallback string selection.
func TestFirstNonEmpty(t *testing.T) {
	t.Run("returns first trimmed value", func(t *testing.T) {
		require.Equal(t, "value", firstNonEmpty("", "  ", "value", "other"))
	})

	t.Run("returns empty string when all values are blank", func(t *testing.T) {
		require.Empty(t, firstNonEmpty("", " "))
	})
}

// TestShortCommit verifies commit hash shortening.
func TestShortCommit(t *testing.T) {
	cases := map[string]string{
		"123456789abcdef": "123456789abc",
		"abc123":          "abc123",
	}
	for value, expected := range cases {
		t.Run(value, func(t *testing.T) {
			require.Equal(t, expected, shortCommit(value))
		})
	}
}
