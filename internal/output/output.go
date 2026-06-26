package output

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// PublicDirName is the project directory copied directly to the output root.
const PublicDirName = "public"

// File is one output file with a safe path relative to the output directory.
type File struct {
	Path    string
	Content []byte
}

// Writer writes output files under a directory.
type Writer struct {
	clean bool
	dir   string
}

// Option configures a Writer.
type Option func(*Writer) error

// New creates a Writer for dir.
func New(dir string, options ...Option) (*Writer, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" || strings.ContainsRune(dir, 0) {
		return nil, ErrDirInvalid
	}

	writer := &Writer{dir: dir}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(writer); err != nil {
			return nil, err
		}
	}

	return writer, nil
}

// WithClean configures whether the output directory is removed before writing.
func WithClean(clean bool) Option {
	return func(writer *Writer) error {
		writer.clean = clean
		return nil
	}
}

// Write writes files to the output directory.
func (writer *Writer) Write(files []File) error {
	if err := writer.prepare(); err != nil {
		return err
	}

	normalized, err := normalizeFiles(files)
	if err != nil {
		return err
	}

	return writer.writeFiles(normalized)
}

// CopyPublic copies files from public/ in projectFiles to the output directory.
func (writer *Writer) CopyPublic(projectFiles fs.FS) error {
	if err := writer.prepare(); err != nil {
		return err
	}

	publicFiles, err := collectPublicFiles(projectFiles)
	if err != nil {
		return err
	}

	return writer.writeFiles(publicFiles)
}

// WriteSite writes rendered files and public assets while detecting collisions.
func (writer *Writer) WriteSite(files []File, projectFiles fs.FS) error {
	if err := writer.prepare(); err != nil {
		return err
	}

	normalized, err := normalizeFiles(files)
	if err != nil {
		return err
	}
	publicFiles, err := collectPublicFiles(projectFiles)
	if err != nil {
		return err
	}

	merged := make([]File, 0, len(normalized)+len(publicFiles))
	merged = append(merged, normalized...)
	merged = append(merged, publicFiles...)
	merged, err = normalizeFiles(merged)
	if err != nil {
		return err
	}

	return writer.writeFiles(merged)
}

// prepare creates or cleans the output directory.
func (writer *Writer) prepare() error {
	if writer == nil || writer.dir == "" {
		return ErrDirInvalid
	}
	if writer.clean {
		if err := os.RemoveAll(writer.dir); err != nil {
			return fmt.Errorf("clean output directory %s: %w", writer.dir, err)
		}
	}
	if err := os.MkdirAll(writer.dir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", writer.dir, err)
	}

	return nil
}

// writeFiles writes already-normalized files.
func (writer *Writer) writeFiles(files []File) error {
	for _, file := range files {
		target := filepath.Join(writer.dir, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create output parent for %s: %w", file.Path, err)
		}
		if err := os.WriteFile(target, file.Content, 0o644); err != nil {
			return fmt.Errorf("write output file %s: %w", file.Path, err)
		}
	}

	return nil
}

// collectPublicFiles returns files from public/ mapped to output-root paths.
func collectPublicFiles(projectFiles fs.FS) ([]File, error) {
	if projectFiles == nil {
		return nil, nil
	}

	files := []File{}
	if err := fs.WalkDir(
		projectFiles,
		PublicDirName,
		func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if name == PublicDirName || entry.IsDir() {
				return nil
			}

			content, err := fs.ReadFile(projectFiles, name)
			if err != nil {
				return fmt.Errorf("read public file %s: %w", name, err)
			}
			outputPath := strings.TrimPrefix(name, PublicDirName+"/")
			files = append(files, File{Content: content, Path: outputPath})
			return nil
		},
	); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("walk public directory %s: %w", PublicDirName, err)
	}

	return normalizeFiles(files)
}

// normalizeFiles validates file paths and detects duplicates.
func normalizeFiles(files []File) ([]File, error) {
	seen := map[string]struct{}{}
	normalized := make([]File, 0, len(files))
	for _, file := range files {
		cleanPath, err := cleanOutputPath(file.Path)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[cleanPath]; exists {
			return nil, fmt.Errorf("%w: %s", ErrPathDuplicate, cleanPath)
		}

		seen[cleanPath] = struct{}{}
		file.Path = cleanPath
		normalized = append(normalized, file)
	}

	return normalized, nil
}

// cleanOutputPath validates a slash-separated output path.
func cleanOutputPath(filePath string) (string, error) {
	rawPath := strings.TrimSpace(filePath)
	if rawPath == "" || strings.ContainsRune(rawPath, 0) || filepath.VolumeName(rawPath) != "" ||
		hasWindowsVolumeName(rawPath) || filepath.IsAbs(rawPath) {
		return "", ErrPathInvalid
	}

	rawPath = strings.ReplaceAll(rawPath, "\\", "/")
	if path.IsAbs(rawPath) || slices.Contains(strings.Split(rawPath, "/"), "..") {
		return "", ErrPathInvalid
	}

	cleanPath := path.Clean(rawPath)
	if cleanPath == "." || strings.HasSuffix(cleanPath, "/") || !fs.ValidPath(cleanPath) {
		return "", ErrPathInvalid
	}

	return cleanPath, nil
}

// hasWindowsVolumeName reports whether a path starts with a Windows drive name.
func hasWindowsVolumeName(name string) bool {
	return len(name) >= 2 && name[1] == ':' &&
		('A' <= name[0] && name[0] <= 'Z' || 'a' <= name[0] && name[0] <= 'z')
}
