package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config controls starter project creation.
type Config struct {
	Force bool
	Root  string
}

// Result describes the files and directories created for a starter project.
type Result struct {
	Directories []string
	Files       []string
	Root        string
}

type fileSpec struct {
	Content string
	Path    string
}

// templateProject contains the embedded starter project tree.
type templateProject struct {
	Directories []string
	Files       []fileSpec
}

const templateRoot = "template"

//go:embed template
var embeddedTemplate embed.FS

// Create writes a starter Veta project to disk.
func Create(config Config) (Result, error) {
	root, err := normalizeRoot(config.Root)
	if err != nil {
		return Result{}, err
	}

	template, err := loadTemplateProject()
	if err != nil {
		return Result{}, err
	}
	directories := template.Directories
	files := template.Files
	if !config.Force {
		existing, err := existingFiles(root, files)
		if err != nil {
			return Result{}, err
		}
		if len(existing) > 0 {
			return Result{}, ExistingFilesError{Paths: existing}
		}
	}

	if err := writeDirectories(root, directories); err != nil {
		return Result{}, err
	}
	if err := writeFiles(root, files); err != nil {
		return Result{}, err
	}

	return Result{Directories: directories, Files: filePaths(files), Root: root}, nil
}

// normalizeRoot returns the cleaned project root.
func normalizeRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	if strings.ContainsRune(root, 0) {
		return "", ErrRootInvalid
	}

	return filepath.Clean(root), nil
}

// loadTemplateProject reads the embedded project template into writable specs.
func loadTemplateProject() (templateProject, error) {
	project := templateProject{}
	err := fs.WalkDir(
		embeddedTemplate,
		templateRoot,
		func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if name == templateRoot {
				return nil
			}

			relativePath := strings.TrimPrefix(name, templateRoot+"/")
			if entry.IsDir() {
				project.Directories = append(project.Directories, relativePath)
				return nil
			}

			content, err := fs.ReadFile(embeddedTemplate, name)
			if err != nil {
				return fmt.Errorf("read project template file %s: %w", name, err)
			}
			project.Files = append(
				project.Files,
				fileSpec{Content: string(content), Path: relativePath},
			)

			return nil
		},
	)
	if err != nil {
		return templateProject{}, fmt.Errorf("load project template: %w", err)
	}

	return project, nil
}

// existingFiles returns starter files that already exist below root.
func existingFiles(root string, files []fileSpec) ([]string, error) {
	existing := []string{}
	for _, file := range files {
		path := filepath.Join(root, filepath.FromSlash(file.Path))
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, file.Path)
			continue
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect starter file %s: %w", path, err)
		}
	}

	return existing, nil
}

// writeDirectories creates all starter directories below root.
func writeDirectories(root string, directories []string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create project root %s: %w", root, err)
	}
	for _, directory := range directories {
		path := filepath.Join(root, filepath.FromSlash(directory))
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create starter directory %s: %w", path, err)
		}
	}

	return nil
}

// writeFiles writes every starter file below root.
func writeFiles(root string, files []fileSpec) error {
	for _, file := range files {
		if err := writeFile(root, file); err != nil {
			return err
		}
	}

	return nil
}

// writeFile writes one starter file below root.
func writeFile(root string, file fileSpec) error {
	path := filepath.Join(root, filepath.FromSlash(file.Path))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create starter file parent %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(file.Content), 0o644); err != nil {
		return fmt.Errorf("write starter file %s: %w", path, err)
	}

	return nil
}

// filePaths returns the slash-separated file paths from specs.
func filePaths(files []fileSpec) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}

	return paths
}
