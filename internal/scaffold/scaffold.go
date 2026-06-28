package scaffold

import (
	"fmt"
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

// Create writes a starter Veta project to disk.
func Create(config Config) (Result, error) {
	root, err := normalizeRoot(config.Root)
	if err != nil {
		return Result{}, err
	}

	directories := starterDirectories()
	files := starterFiles()
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

// starterDirectories returns the standard Veta project directories.
func starterDirectories() []string {
	return []string{
		"components",
		"data",
		"filters",
		"pages",
		"public",
		"templates",
	}
}

// starterFiles returns the starter project file set.
func starterFiles() []fileSpec {
	return []fileSpec{
		{Path: "veta.yaml", Content: starterConfig},
		{Path: "data/site.json", Content: starterData},
		{Path: "pages/site.js", Content: starterPages},
		{Path: "templates/base.pongo", Content: starterTemplate},
		{Path: "components/card.pongo", Content: starterComponent},
		{Path: "filters/uppercase.js", Content: starterFilter},
		{Path: "public/styles.css", Content: starterStyles},
		{Path: "public/robots.txt", Content: starterRobots},
	}
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

const starterConfig = `build:
  output: dist
  clean: true
  debug: false
tailwindcss:
  input: public/styles.css
  output: styles.css
  minify: true
`

const starterData = `{
  "title": "Veta Starter",
  "description": "A small site generated with Veta."
}
`

const starterPages = "export default function({ data }) {\n" +
	"  return [\n" +
	"    {\n" +
	"      permalink: \"/\",\n" +
	"      template: \"base\",\n" +
	"      title: data.site.title,\n" +
	"      navOrder: 1,\n" +
	"      content: \"<card>Build something fast with **Veta**.</card>\",\n" +
	"    },\n" +
	"    {\n" +
	"      permalink: \"/about/\",\n" +
	"      template: \"base\",\n" +
	"      title: \"About\",\n" +
	"      navOrder: 2,\n" +
	"      content: \"# About\\n\\nThis page was generated from pages/site.js.\",\n" +
	"    },\n" +
	"  ];\n" +
	"}\n"

const starterTemplate = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="description" content="{{ data.site.description }}">
    <title>{{ page.title }} - {{ data.site.title }}</title>
    <link rel="stylesheet" href="/styles.css">
  </head>
  <body class="min-h-screen bg-zinc-950 text-zinc-100">
    <header class="border-b border-white/10 px-6 py-4">
      <nav class="mx-auto flex max-w-5xl gap-4">
        {% for item in pages %}
          <a class="text-sm text-zinc-300 hover:text-white" href="{{ item.permalink }}">{{ item.title }}</a>
        {% endfor %}
      </nav>
    </header>
    <main class="mx-auto max-w-5xl px-6 py-16">
      {{ page.content }}
    </main>
  </body>
</html>
`

const starterComponent = `<section class="rounded-2xl border border-white/10 bg-white/5 p-8 shadow-2xl shadow-black/30">
  {{ props.content }}
</section>
`

const starterFilter = `export default function(input) {
  return String(input).toUpperCase();
}
`

const starterStyles = `@import "tailwindcss";
`

const starterRobots = `User-agent: *
Allow: /
`
