package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
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
}

func TestRunBuildHelp(t *testing.T) {
	var stderr bytes.Buffer

	err := Run(context.Background(), []string{"build", "--help"}, nil, &stderr)
	require.NoError(t, err)
	require.Contains(t, stderr.String(), "veta build")
	require.Contains(t, stderr.String(), "--root")
}

func TestRunErrors(t *testing.T) {
	err := Run(context.Background(), []string{"serve"}, nil, nil)
	require.ErrorIs(t, err, ErrUnknownCommand)

	err = Run(context.Background(), []string{"build", "unexpected"}, nil, nil)
	require.ErrorIs(t, err, ErrUnknownCommand)
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
	writeCLIFile(t, root, "templates/base.pongo", `{{ content }}`)
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
