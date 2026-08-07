---
title: "Config Reference"
description: "Complete reference for veta.yaml configuration fields."
---

# Config Reference

Veta config is YAML. Unknown fields are rejected.

## Example

```yaml
build:
  output: dist
  clean: true

html:
  minify: true

dev:
  host: 127.0.0.1
  port: 3000
  watch:
    - content

tailwindcss:
  stylesheets:
    - styles.css
  minify: true

theme:
  source: "./theme"
```

## `build.output`

Type: string

Default: `dist`

The production output directory used by `veta build`.

Must be a relative project path.

## `build.clean`

Type: boolean

Default: `false`

When true, Veta removes the output directory before writing the build.

## `html.minify`

Type: boolean

Default: `false`

When true, minifies generated `.html` files only.

## `dev.host`

Type: string

Default: `127.0.0.1`

The local interface used by `veta dev`.

## `dev.port`

Type: number

Default: `3000`

The TCP port used by `veta dev`.

## `dev.watch`

Type: array of strings

Default: `[]`

Additional project-relative files or directories watched by `veta dev`. Directories are watched recursively. These paths are added to Veta's built-in watch set.

Example:

```yaml
dev:
  watch:
    - content
    - docs
```

## `tailwindcss.stylesheets`

Type: array of strings

Default: `[]`

When set, enables Tailwind CSS. Each path is relative to `public/`.

Example:

```yaml
tailwindcss:
  stylesheets:
    - styles.css
    - admin.css
```

This reads `public/styles.css` and `public/admin.css`, then writes generated CSS to `dist/styles.css` and `dist/admin.css`.

## `tailwindcss.minify`

Type: boolean

Default: `false`

When true, passes Tailwind's minification flag to the embedded Tailwind CSS standalone CLI.

## `theme.source`

Type: string

Default: empty

When set, resolves and composes a theme with the project.

Examples:

```yaml
theme:
  source: "./themes/blog"
```

```yaml
theme:
  source: "owner/veta-theme-name@v1.0.0"
```

> Note: The theme should match the owner/veta-theme-{name}@{ref} pattern.

## Supported File Names

Veta checks these names in order:

```txt
veta.yaml
veta.yml
.veta.yaml
.veta.yml
```
