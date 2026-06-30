package dev

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/varavelio/veta/internal/config"
)

var watchedDirectories = []string{
	"pages",
	"data",
	"templates",
	"components",
	"filters",
	"public",
}

type fileFingerprint struct {
	isDir   bool
	mode    fs.FileMode
	modTime time.Time
	size    int64
}

type fileSnapshot map[string]fileFingerprint

type snapshotFunc func() (fileSnapshot, error)

// defaultWatchPaths returns the project files and directories watched by veta dev.
func defaultWatchPaths() []string {
	paths := config.FileNames()
	paths = append(paths, watchedDirectories...)
	return paths
}

// watchProject polls project files and emits debounced change notifications.
func watchProject(
	ctx context.Context,
	root string,
	paths []string,
	interval time.Duration,
	debounce time.Duration,
) (<-chan struct{}, <-chan error) {
	return watchSnapshots(ctx, func() (fileSnapshot, error) {
		return snapshotPaths(root, paths)
	}, interval, debounce)
}

// watchSnapshots polls snapshots and emits changes after the debounce period.
func watchSnapshots(
	ctx context.Context,
	takeSnapshot snapshotFunc,
	interval time.Duration,
	debounce time.Duration,
) (<-chan struct{}, <-chan error) {
	if interval <= 0 {
		interval = defaultPollInterval
	}

	changes := make(chan struct{}, 1)
	errors := make(chan error, 1)

	go func() {
		defer close(changes)
		defer close(errors)

		previous, err := takeSnapshot()
		if err != nil {
			sendWatcherError(ctx, errors, err)
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var debounceTimer *time.Timer
		var debounceReady <-chan time.Time
		defer func() {
			stopTimer(debounceTimer)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current, err := takeSnapshot()
				if err != nil {
					sendWatcherError(ctx, errors, err)
					return
				}
				if previous.equal(current) {
					continue
				}

				previous = current
				debounceTimer, debounceReady = resetDebounceTimer(debounceTimer, debounce)
			case <-debounceReady:
				debounceReady = nil
				sendChange(changes)
			}
		}
	}()

	return changes, errors
}

// snapshotPaths captures metadata for the watched paths that currently exist.
func snapshotPaths(root string, paths []string) (fileSnapshot, error) {
	snapshot := fileSnapshot{}
	for _, watchPath := range paths {
		watchPath = strings.TrimSpace(watchPath)
		if watchPath == "" {
			continue
		}

		target := filepath.Join(root, filepath.FromSlash(watchPath))
		info, err := os.Lstat(target)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, fmt.Errorf("inspect watched path %s: %w", target, err)
		}

		if !info.IsDir() {
			if err := snapshot.add(root, target, info); err != nil {
				return nil, err
			}
			continue
		}

		if err := filepath.WalkDir(target, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			info, err := entry.Info()
			if err != nil {
				return err
			}

			return snapshot.add(root, path, info)
		}); err != nil {
			return nil, fmt.Errorf("walk watched path %s: %w", target, err)
		}
	}

	return snapshot, nil
}

// add records one filesystem entry relative to root.
func (snapshot fileSnapshot) add(root, path string, info fs.FileInfo) error {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return fmt.Errorf("resolve watched path %s: %w", path, err)
	}

	snapshot[filepath.ToSlash(relativePath)] = fileFingerprint{
		isDir:   info.IsDir(),
		mode:    info.Mode(),
		modTime: info.ModTime(),
		size:    info.Size(),
	}
	return nil
}

// equal reports whether two snapshots contain the same filesystem metadata.
func (snapshot fileSnapshot) equal(other fileSnapshot) bool {
	if len(snapshot) != len(other) {
		return false
	}

	for path, state := range snapshot {
		otherState, ok := other[path]
		if !ok {
			return false
		}
		if state.isDir != otherState.isDir || state.mode != otherState.mode ||
			state.size != otherState.size || !state.modTime.Equal(otherState.modTime) {
			return false
		}
	}

	return true
}

// resetDebounceTimer starts or restarts the debounce timer.
func resetDebounceTimer(timer *time.Timer, debounce time.Duration) (*time.Timer, <-chan time.Time) {
	if debounce <= 0 {
		debounce = time.Nanosecond
	}
	if timer == nil {
		timer = time.NewTimer(debounce)
		return timer, timer.C
	}

	stopTimer(timer)
	timer.Reset(debounce)
	return timer, timer.C
}

// stopTimer stops a timer and drains it when needed.
func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

// sendChange emits a coalesced change notification without blocking polling.
func sendChange(changes chan<- struct{}) {
	select {
	case changes <- struct{}{}:
	default:
	}
}

// sendWatcherError emits a watcher error unless the watcher is already stopped.
func sendWatcherError(ctx context.Context, errors chan<- error, err error) {
	select {
	case errors <- err:
	case <-ctx.Done():
	}
}
