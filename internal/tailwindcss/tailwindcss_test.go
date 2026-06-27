package tailwindcss

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	cacheDir := t.TempDir()
	binary := fakeBinary(t)

	file, err := Build(
		context.Background(),
		fstest.MapFS{
			"styles/app.css":       {Data: []byte(`@import "tailwindcss";`)},
			"templates/base.pongo": {Data: []byte(`<div class="text-red-500"></div>`)},
		},
		[]Document{{Content: []byte(`<main class="text-blue-500"></main>`), Path: "index.html"}},
		Config{Input: "styles/app.css", Minify: true, Output: "assets/app.css"},
		WithBinary(binary),
		WithCacheDir(cacheDir),
	)
	require.NoError(t, err)
	require.Equal(t, "assets/app.css", file.Path)
	require.Contains(t, string(file.Content), "minify=true")
	require.Contains(t, string(file.Content), "rendered=true")
	require.FileExists(t, filepath.Join(cacheDir, Version, binary.Name))
}

func TestBuildRewritesInvalidCachedBinary(t *testing.T) {
	cacheDir := t.TempDir()
	binary := fakeBinary(t)
	cachedPath := filepath.Join(cacheDir, Version, binary.Name)
	require.NoError(t, os.MkdirAll(filepath.Dir(cachedPath), 0o755))
	require.NoError(t, os.WriteFile(cachedPath, []byte("bad"), 0o755))

	_, err := Build(
		context.Background(),
		fstest.MapFS{"styles/app.css": {Data: []byte(`@import "tailwindcss";`)}},
		nil,
		Config{Input: "styles/app.css", Output: "assets/app.css"},
		WithBinary(binary),
		WithCacheDir(cacheDir),
	)
	require.NoError(t, err)

	content, err := os.ReadFile(cachedPath)
	require.NoError(t, err)
	require.Equal(t, binary.Content, content)
}

func TestBuildErrors(t *testing.T) {
	_, err := Build(context.Background(), nil, nil, Config{})
	require.ErrorIs(t, err, ErrConfigInvalid)

	_, err = Build(
		context.Background(),
		fstest.MapFS{},
		nil,
		Config{Input: "", Output: "app.css"},
		WithExecutablePath("tailwindcss"),
	)
	require.ErrorIs(t, err, ErrConfigInvalid)

	_, err = Build(
		context.Background(),
		fstest.MapFS{},
		nil,
		Config{Input: "../app.css", Output: "app.css"},
		WithExecutablePath("tailwindcss"),
	)
	require.ErrorIs(t, err, ErrConfigInvalid)

	_, err = Build(
		context.Background(),
		fstest.MapFS{},
		nil,
		Config{Input: "app.css", Output: "app.css"},
		WithCacheDir(""),
	)
	require.ErrorIs(t, err, ErrCacheDirInvalid)
}

func TestRunFailure(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "tailwind-fail-*")
	require.NoError(t, err)
	_, err = file.Write(failingBinaryContent())
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.NoError(t, os.Chmod(file.Name(), 0o755))

	_, err = Build(
		context.Background(),
		fstest.MapFS{"styles/app.css": {Data: []byte(`@import "tailwindcss";`)}},
		nil,
		Config{Input: "styles/app.css", Output: "assets/app.css"},
		WithExecutablePath(file.Name()),
	)
	require.ErrorIs(t, err, ErrRunFailed)
}

func fakeBinary(t *testing.T) Binary {
	t.Helper()

	content := fakeBinaryContent()
	hash := sha256.Sum256(content)
	name := "tailwindcss-fake"
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}

	return Binary{Content: content, Name: name, SHA256: hex.EncodeToString(hash[:])}
}

func fakeBinaryContent() []byte {
	if runtime.GOOS == "windows" {
		return []byte(strings.Join([]string{
			"@echo off",
			"set out=",
			"set minify=false",
			":loop",
			"if \"%1\"==\"\" goto done",
			"if \"%1\"==\"-o\" set out=%2& shift& shift& goto loop",
			"if \"%1\"==\"--minify\" set minify=true& shift& goto loop",
			"shift",
			"goto loop",
			":done",
			"if exist veta-rendered\\index.html (set rendered=true) else (set rendered=false)",
			"> \"%out%\" echo minify=%minify% rendered=%rendered%",
		}, "\r\n"))
	}

	return []byte(strings.Join([]string{
		"#!/bin/sh",
		"out=",
		"minify=false",
		"while [ $# -gt 0 ]; do",
		"  case \"$1\" in",
		"    -o) out=\"$2\"; shift 2 ;;",
		"    --minify) minify=true; shift ;;",
		"    *) shift ;;",
		"  esac",
		"done",
		"if [ -f veta-rendered/index.html ]; then rendered=true; else rendered=false; fi",
		"printf 'minify=%s rendered=%s\\n' \"$minify\" \"$rendered\" > \"$out\"",
	}, "\n"))
}

func failingBinaryContent() []byte {
	if runtime.GOOS == "windows" {
		return []byte("@echo off\r\necho broken 1>&2\r\nexit /b 1\r\n")
	}

	return []byte("#!/bin/sh\necho broken >&2\nexit 1\n")
}
