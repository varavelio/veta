---
title: "Build And Output"
description: "Understand production builds, output paths, cleaning, generated HTML minification, and public assets."
---

# Build And Output

Use `veta build` to create a production static site:

```sh
veta build
```

Veta discovers the config file, derives the project root, loads data and pages, renders documents, writes generated files, copies public assets, and optionally runs Tailwind CSS.

## Output Directory

Configure the output directory in `veta.yaml`:

```yaml
build:
  output: dist
```

The path must be relative to the project.

## Clean Builds

```yaml
build:
  clean: true
```

When enabled, Veta removes the output directory before writing new files. This avoids stale files from previous builds.

## Generated Files

Page permalinks determine generated file paths:

```txt
/                  -> dist/index.html
/about/            -> dist/about/index.html
/feed.xml          -> dist/feed.xml
/llms.txt          -> dist/llms.txt
```

Veta validates output paths and rejects duplicate output files.

## Public Assets

Files under `public/` are copied to the output root after generated files are prepared:

```txt
public/robots.txt  -> dist/robots.txt
public/logo.svg    -> dist/logo.svg
```

If a generated file and a public file claim the same output path, Veta fails the build.

## HTML Minification

```yaml
html:
  minify: true
```

This minifies generated files with a `.html` extension. It does not minify:

- `.xml`
- `.md`
- `.json`
- `.txt`
- `.js`
- `.css`
- copied files from `public/`

## Tailwind CSS

When configured, Tailwind CSS runs after Veta writes the generated site and public files:

```yaml
tailwindcss:
  stylesheet: styles.css
  minify: true
```

This allows Tailwind to scan the generated output and include classes used by templates, components, and page content.

## Explicit Config File

Build with an explicit config:

```sh
veta build --config ./config/veta.yaml
```

The project root becomes the directory containing that config file.
