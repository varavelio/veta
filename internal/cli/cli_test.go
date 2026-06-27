package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/veta/internal/scaffold"
)

func TestRunBuildCommand(t *testing.T) {
	root := newCLISite(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(
		context.Background(),
		[]string{"build", "--root", root, "--out", "public-build", "--clean"},
		&stdout,
		&stderr,
	)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Built 1 page(s)")
	require.FileExists(t, filepath.Join(root, "public-build", "index.html"))
	require.Empty(t, stderr.String())
}

func TestRunBuildAliasFlags(t *testing.T) {
	root := newCLISite(t)
	var stdout bytes.Buffer

	err := Run(context.Background(), []string{"--root", root, "--out", "dist"}, &stdout, nil)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Built 1 page(s)")
	require.FileExists(t, filepath.Join(root, "dist", "index.html"))
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer

	err := Run(context.Background(), []string{"--help"}, &stdout, nil)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Usage:")
	require.Contains(t, stdout.String(), "build")
	require.Contains(t, stdout.String(), "init")
	require.Contains(t, stdout.String(), "version")
}

func TestRunBuildHelp(t *testing.T) {
	var stdout bytes.Buffer

	err := Run(context.Background(), []string{"build", "--help"}, &stdout, nil)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "veta build")
	require.Contains(t, stdout.String(), "--root")
}

func TestRunVersion(t *testing.T) {
	for _, flag := range []string{"--version", "-v", "version"} {
		t.Run(flag, func(t *testing.T) {
			var stdout bytes.Buffer

			err := Run(context.Background(), []string{flag}, &stdout, nil)
			require.NoError(t, err)
			require.Equal(t, "veta dev\n", stdout.String())
		})
	}
}

func TestRunInitCommand(t *testing.T) {
	root := filepath.Join(t.TempDir(), "site")
	var stdout bytes.Buffer

	err := Run(context.Background(), []string{"init", root}, &stdout, nil)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Initialized Veta project")

	for _, directory := range []string{
		"components",
		"data",
		"filters",
		"pages",
		"public",
		"styles",
		"templates",
	} {
		require.DirExists(t, filepath.Join(root, directory))
	}
	for _, file := range []string{
		"veta.yaml",
		"data/site.json",
		"pages/site.js",
		"templates/base.pongo",
		"components/card.pongo",
		"filters/uppercase.js",
		"styles/app.css",
		"public/robots.txt",
	} {
		require.FileExists(t, filepath.Join(root, filepath.FromSlash(file)))
	}
}

func TestRunInitRefusesOverwrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "site")
	require.NoError(t, Run(context.Background(), []string{"init", root}, nil, nil))

	var stderr bytes.Buffer
	err := Run(context.Background(), []string{"init", root}, nil, &stderr)
	require.ErrorIs(t, err, scaffold.ErrFileExists)
	require.Contains(t, stderr.String(), "Cannot initialize the project")
	require.Contains(t, stderr.String(), "veta init --force")
}

func TestRunErrors(t *testing.T) {
	var stderr bytes.Buffer
	err := Run(context.Background(), []string{"serve"}, nil, &stderr)
	require.ErrorIs(t, err, ErrUsage)
	require.Contains(t, stderr.String(), "error:")
	require.Contains(t, stderr.String(), "Usage:")

	stderr.Reset()
	err = Run(context.Background(), []string{"build", "unexpected"}, nil, &stderr)
	require.ErrorIs(t, err, ErrUsage)
	require.Contains(t, stderr.String(), "error:")
	require.Contains(t, stderr.String(), "veta build")
}

func TestRunContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Run(ctx, []string{"build"}, nil, nil)
	require.True(t, errors.Is(err, context.Canceled))
}

func newCLISite(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeCLIFile(t, root, "templates/base.pongo", `{{ page.content }}`)
	writeCLIFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", layout: "templates/base", content: "Hello" }];
}
`)

	return root
}

func writeCLIFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
