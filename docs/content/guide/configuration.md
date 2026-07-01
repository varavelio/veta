---
title: "Configuration"
description: "Understand veta.yaml, config discovery, build settings, HTML minification, Tailwind CSS, and themes."
---

# Configuration

Veta configuration lives in YAML. The supported file names are checked in this order:

```txt
veta.yaml
veta.yml
.veta.yaml
.veta.yml
```

When you run `veta build` or `veta dev`, Veta searches from the current directory upward through parent directories until it finds one of those files. You can also pass an explicit config file:

```sh
veta build --config path/to/veta.yaml
veta dev --config path/to/veta.yaml
```

The project root is the directory that contains the resolved config file.

## Minimal Config

```yaml
build:
  output: dist
```

If `build.output` is omitted or blank, Veta uses `dist`.

## Full Common Config

```yaml
build:
  output: dist
  clean: true
  debug: false

html:
  minify: true

tailwindcss:
  stylesheet: styles.css
  minify: true

theme:
  source: "./theme"
```

## `build`

`build` contains build workflow settings.

```yaml
build:
  output: dist
  clean: true
  debug: false
```

`output` is the production output directory used by `veta build`. It must be a relative project path.

`clean` removes the output directory before writing a new build.

`debug` disables template caching and prints additional debug details where applicable.

## `html`

`html` contains generated HTML settings.

```yaml
html:
  minify: true
```

`html.minify` minifies generated `.html` files only. It does not minify XML, Markdown, JSON, text, JavaScript, CSS, or files copied from `public/`.

## `tailwindcss`

`tailwindcss` enables Veta's embedded Tailwind CSS standalone integration.

```yaml
tailwindcss:
  stylesheet: styles.css
  minify: true
```

`stylesheet` is relative to `public/`. With the config above, Veta reads `public/styles.css` and writes the generated CSS to `dist/styles.css`.

`minify` passes Tailwind's minification flag to the standalone CLI.

If `tailwindcss.stylesheet` is omitted, Tailwind CSS does not run.

## `theme`

`theme.source` points to a local theme directory or a GitHub theme source.

```yaml
theme:
  source: "./themes/clean"
```

Themes can provide `templates/`, `components/`, `filters/`, `data/`, and `public/`. Project files override theme files.

## Unknown Fields

Veta rejects unknown config fields. This catches typos early and keeps configuration predictable.
