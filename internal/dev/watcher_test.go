package dev

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSnapshotPathsDetectsProjectChanges(t *testing.T) {
	root := t.TempDir()
	writeDevTestFile(t, root, "veta.yaml", "build:\n  output: dist\n")
	writeDevTestFile(t, root, "pages/site.js", "export default function() { return []; }\n")

	before, err := snapshotPaths(root, []string{"veta.yaml", "pages"})
	require.NoError(t, err)
	require.Contains(t, before, "veta.yaml")
	require.Contains(t, before, "pages/site.js")

	writeDevTestFile(t, root, "pages/extra.js", "export default function() { return []; }\n")
	after, err := snapshotPaths(root, []string{"veta.yaml", "pages"})
	require.NoError(t, err)
	require.False(t, before.equal(after))

	require.NoError(t, os.Remove(filepath.Join(root, "pages", "extra.js")))
	finalSnapshot, err := snapshotPaths(root, []string{"veta.yaml", "pages"})
	require.NoError(t, err)
	require.NotContains(t, finalSnapshot, "pages/extra.js")
}

func TestWatchSnapshotsDebouncesRapidChanges(t *testing.T) {
	snapshots := []fileSnapshot{
		{"pages/site.js": {size: 1}},
		{"pages/site.js": {size: 2}},
		{"pages/site.js": {size: 3}},
		{"pages/site.js": {size: 3}},
	}
	var calls atomic.Int64
	snapshot := func() (fileSnapshot, error) {
		index := int(calls.Add(1)) - 1
		if index >= len(snapshots) {
			index = len(snapshots) - 1
		}

		return snapshots[index], nil
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	changes, errors := watchSnapshots(ctx, snapshot, 5*time.Millisecond, 30*time.Millisecond)

	select {
	case <-changes:
	case err := <-errors:
		require.NoError(t, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for debounced change")
	}

	select {
	case <-changes:
		t.Fatal("received duplicate change despite debounce")
	case err := <-errors:
		require.NoError(t, err)
	case <-time.After(60 * time.Millisecond):
	}
}

func TestWatchPathsIncludesExplicitConfigFile(t *testing.T) {
	root := t.TempDir()
	configFile := filepath.Join(root, "custom.yaml")
	server := server{config: Config{ConfigFile: configFile}}

	paths := server.watchPaths(root)

	require.Contains(t, paths, "custom.yaml")
}
