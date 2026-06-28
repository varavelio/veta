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
		[]string{"build", "--config", filepath.Join(root, "veta.yaml")},
		&stdout,
		&stderr,
	)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Built 1 page(s)")
	require.FileExists(t, filepath.Join(root, "dist", "index.html"))
	require.Empty(t, stderr.String())
}

func TestRunDefaultShowsHelp(t *testing.T) {
	var stdout bytes.Buffer

	err := Run(context.Background(), nil, &stdout, nil)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Version:")
	require.Contains(t, stdout.String(), "Commit:")
	require.Contains(t, stdout.String(), "Repository:")
	require.Contains(t, stdout.String(), "https://github.com/varavelio/veta")
	require.Contains(t, stdout.String(), "Usage:")
	require.Contains(t, stdout.String(), "veta build")
}

func TestRunBuildDiscoversConfig(t *testing.T) {
	root := newCLISite(t)
	child := filepath.Join(root, "content", "docs")
	require.NoError(t, os.MkdirAll(child, 0o755))
	t.Chdir(child)
	var stdout bytes.Buffer

	err := Run(context.Background(), []string{"build"}, &stdout, nil)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Built 1 page(s)")
	require.FileExists(t, filepath.Join(root, "dist", "index.html"))
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer

	err := Run(context.Background(), []string{"--help"}, &stdout, nil)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Version:")
	require.Contains(t, stdout.String(), "Commit:")
	require.Contains(t, stdout.String(), "Repository:")
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
	require.Contains(t, stdout.String(), "--config")
	require.NotContains(t, stdout.String(), "--root")
	require.NotContains(t, stdout.String(), "--out")
	require.NotContains(t, stdout.String(), "--clean")
	require.NotContains(t, stdout.String(), "--debug")
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
	require.Contains(t, stdout.String(), "Next steps:")
	require.Contains(t, stdout.String(), "veta build")
	require.Contains(t, stdout.String(), "Build settings live in veta.yaml")

	for _, directory := range []string{
		"components",
		"data",
		"filters",
		"pages",
		"public",
		"templates",
	} {
		require.DirExists(t, filepath.Join(root, directory))
	}
	for _, file := range []string{
		"veta.yaml",
		"data/site.json",
		"pages/site.js",
		"templates/base.pongo",
		"components/note.pongo",
		"filters/label.js",
		"public/styles.css",
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
	require.Contains(t, stderr.String(), "error")
	require.Contains(t, stderr.String(), "Usage:")

	stderr.Reset()
	err = Run(context.Background(), []string{"build", "unexpected"}, nil, &stderr)
	require.ErrorIs(t, err, ErrUsage)
	require.Contains(t, stderr.String(), "error")
	require.Contains(t, stderr.String(), "veta build")

	stderr.Reset()
	err = Run(context.Background(), []string{"--config", "veta.yaml"}, nil, &stderr)
	require.ErrorIs(t, err, ErrUsage)
	require.Contains(t, stderr.String(), "unknown argument")
}

func TestRunConfigNotFoundError(t *testing.T) {
	t.Chdir(t.TempDir())
	var stderr bytes.Buffer

	err := Run(context.Background(), []string{"build"}, nil, &stderr)
	require.Error(t, err)
	require.Contains(t, stderr.String(), "Could not find a Veta config file")
	require.Contains(t, stderr.String(), "veta init")
	require.Contains(t, stderr.String(), "veta build --config ./veta.yaml")
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
	writeCLIFile(t, root, "veta.yaml", `
build:
  output: dist
  clean: true
`)
	writeCLIFile(t, root, "templates/base.pongo", `{{ page.content }}`)
	writeCLIFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", template: "base", content: "Hello" }];
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
