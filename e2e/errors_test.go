//go:build e2e

package e2e

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReportsHumanErrors verifies important failure modes through the CLI.
func TestReportsHumanErrors(t *testing.T) {
	t.Run("missing config explains how to start", func(t *testing.T) {
		result := runVeta(t, t.TempDir(), "build")
		result.requireFailure(t)

		require.Contains(t, result.stderr, "Could not find a Veta config file.")
		require.Contains(t, result.stderr, "veta init")
		require.Contains(t, result.stderr, "veta build --config ./veta.yaml")
	})

	t.Run("duplicate output paths point at pages", func(t *testing.T) {
		projectRoot := t.TempDir()
		writeProjectFile(t, projectRoot, "veta.yaml", "build:\n  clean: true\n")
		writeProjectFile(t, projectRoot, "pages/site.js", `
export default function() {
  return [
    { permalink: "/", layout: "templates/plain", content: "One" },
    { permalink: "/", layout: "templates/plain", content: "Two" },
  ];
}
`)

		result := runVeta(t, projectRoot, "build")
		result.requireFailure(t)

		require.Contains(t, result.stderr, "Page generation failed.")
		require.Contains(t, result.stderr, "pages/")
		require.Contains(t, result.stderr, "duplicate")
	})

	t.Run("init refuses to overwrite starter files", func(t *testing.T) {
		workspace := t.TempDir()
		projectRoot := filepath.Join(workspace, "site")
		runVeta(t, workspace, "init", projectRoot).requireSuccess(t)

		result := runVeta(t, workspace, "init", projectRoot)
		result.requireFailure(t)

		require.Contains(t, result.stderr, "Cannot initialize the project")
		require.Contains(t, result.stderr, "veta.yaml")
		require.Contains(t, result.stderr, "veta init --force")
	})
}
