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
	workDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(workDir, "index.html"),
		[]byte(`<main class="text-blue-500"></main>`),
		0o644,
	))

	err := Build(
		context.Background(),
		fstest.MapFS{
			"public/styles.css": {Data: []byte(`@import "tailwindcss";`)},
			"templates/base.j2": {Data: []byte(`<div class="text-red-500"></div>`)},
		},
		Config{
			Input:   "public/styles.css",
			Minify:  true,
			Output:  "assets/app.css",
			WorkDir: workDir,
		},
		WithBinary(binary),
		WithCacheDir(cacheDir),
	)
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(workDir, "assets", "app.css"))
	require.NoError(t, err)
	require.Contains(t, string(content), "minify=true")
	require.Contains(t, string(content), "rendered=true")
	require.Contains(t, string(content), "input=true")
	require.FileExists(t, filepath.Join(cacheDir, Version, binary.Name))
}

func TestBuildRewritesInvalidCachedBinary(t *testing.T) {
	cacheDir := t.TempDir()
	binary := fakeBinary(t)
	cachedPath := filepath.Join(cacheDir, Version, binary.Name)
	require.NoError(t, os.MkdirAll(filepath.Dir(cachedPath), 0o755))
	require.NoError(t, os.WriteFile(cachedPath, []byte("bad"), 0o755))

	err := Build(
		context.Background(),
		fstest.MapFS{"public/styles.css": {Data: []byte(`@import "tailwindcss";`)}},
		Config{Input: "public/styles.css", Output: "assets/app.css", WorkDir: t.TempDir()},
		WithBinary(binary),
		WithCacheDir(cacheDir),
	)
	require.NoError(t, err)

	content, err := os.ReadFile(cachedPath)
	require.NoError(t, err)
	require.Equal(t, binary.Content, content)
}

func TestBuildErrors(t *testing.T) {
	err := Build(context.Background(), nil, Config{})
	require.ErrorIs(t, err, ErrConfigInvalid)

	err = Build(
		context.Background(),
		fstest.MapFS{},
		Config{Input: "", Output: "app.css", WorkDir: t.TempDir()},
		WithExecutablePath("tailwindcss"),
	)
	require.ErrorIs(t, err, ErrConfigInvalid)

	err = Build(
		context.Background(),
		fstest.MapFS{},
		Config{Input: "../app.css", Output: "app.css", WorkDir: t.TempDir()},
		WithExecutablePath("tailwindcss"),
	)
	require.ErrorIs(t, err, ErrConfigInvalid)

	err = Build(
		context.Background(),
		fstest.MapFS{},
		Config{Input: "app.css", Output: "", WorkDir: t.TempDir()},
		WithExecutablePath("tailwindcss"),
	)
	require.ErrorIs(t, err, ErrConfigInvalid)

	err = Build(
		context.Background(),
		fstest.MapFS{},
		Config{Input: "app.css", Output: "app.css", WorkDir: ""},
		WithExecutablePath("tailwindcss"),
	)
	require.ErrorIs(t, err, ErrConfigInvalid)

	err = Build(
		context.Background(),
		fstest.MapFS{},
		Config{Input: "app.css", Output: "app.css", WorkDir: t.TempDir()},
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

	err = Build(
		context.Background(),
		fstest.MapFS{"public/styles.css": {Data: []byte(`@import "tailwindcss";`)}},
		Config{Input: "public/styles.css", Output: "assets/app.css", WorkDir: t.TempDir()},
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
			"set in=",
			"set out=",
			"set minify=false",
			":loop",
			"if \"%1\"==\"\" goto done",
			"if \"%1\"==\"-i\" set in=%2& shift& shift& goto loop",
			"if \"%1\"==\"-o\" set out=%2& shift& shift& goto loop",
			"if \"%1\"==\"--minify\" set minify=true& shift& goto loop",
			"shift",
			"goto loop",
			":done",
			"if exist index.html (set rendered=true) else (set rendered=false)",
			"set input=false",
			"if not \"%in%\"==\"\" findstr /C:\"tailwindcss\" \"%in%\" >nul && set input=true",
			"> \"%out%\" echo minify=%minify% rendered=%rendered% input=%input%",
		}, "\r\n"))
	}

	return []byte(strings.Join([]string{
		"#!/bin/sh",
		"in=",
		"out=",
		"minify=false",
		"while [ $# -gt 0 ]; do",
		"  case \"$1\" in",
		"    -i) in=\"$2\"; shift 2 ;;",
		"    -o) out=\"$2\"; shift 2 ;;",
		"    --minify) minify=true; shift ;;",
		"    *) shift ;;",
		"  esac",
		"done",
		"if [ -f index.html ]; then rendered=true; else rendered=false; fi",
		"if [ -n \"$in\" ] && grep -q 'tailwindcss' \"$in\"; then input=true; else input=false; fi",
		"printf 'minify=%s rendered=%s input=%s\\n' \"$minify\" \"$rendered\" \"$input\" > \"$out\"",
	}, "\n"))
}

func failingBinaryContent() []byte {
	if runtime.GOOS == "windows" {
		return []byte("@echo off\r\necho broken 1>&2\r\nexit /b 1\r\n")
	}

	return []byte("#!/bin/sh\necho broken >&2\nexit 1\n")
}
