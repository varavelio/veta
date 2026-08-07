---
title: "Assets And Tailwind CSS"
description: "Use public assets and Veta's embedded Tailwind CSS standalone integration."
---

# Assets And Tailwind CSS

Static assets live in `public/`. Tailwind CSS is configured through `veta.yaml` and uses one or more stylesheets inside `public/` as entrypoints.

## Public Assets

Files in `public/` are copied to the output root:

```txt
public/robots.txt           -> dist/robots.txt
public/images/logo.svg      -> dist/images/logo.svg
```

Public files are copied as-is. Veta does not minify or transform copied public assets.

Use the `url` template function when linking copied assets from templates. It returns a path relative to the current page, so the generated site can be served from a domain root, a subdirectory, or static storage without a configured base URL:

```html
<link rel="stylesheet" href="{{ url("/styles.css") }}">
<img src="{{ url("/images/logo.svg") }}" alt="Logo">
```

## Tailwind CSS Entrypoints

The starter uses `public/styles.css`:

```css
@import "tailwindcss";
```

Configure it with:

```yaml
tailwindcss:
  stylesheets:
    - styles.css
  minify: true
```

`stylesheets` entries are relative to `public/`, so `styles.css` means `public/styles.css`.

You can configure multiple entrypoints:

```yaml
tailwindcss:
  stylesheets:
    - styles.css
    - admin.css
  minify: true
```

## Generated CSS Output

Veta writes each compiled stylesheet to the build output using the same path:

```txt
public/styles.css           -> dist/styles.css
public/admin.css            -> dist/admin.css
```

Tailwind scans the materialized output directory, so classes used in generated HTML are included.

If an entrypoint should scan a narrower set of files, configure that in the CSS file with Tailwind's `@source` directive.

## Minification

`tailwindcss.minify: true` minifies the generated stylesheet through Tailwind CSS.

This setting is separate from `html.minify`, which only affects generated `.html` files.

## Disabling Tailwind CSS

Remove `tailwindcss.stylesheets` or leave it empty:

```yaml
tailwindcss:
  stylesheets: []
  minify: true
```

Without `stylesheets`, Veta does not run Tailwind CSS.

## Practical Pattern

Use `public/styles.css` for your Tailwind entrypoint and keep images, fonts, and static files under `public/`:

```txt
public/
  styles.css
  fonts/inter.woff2
  images/logo.svg
  robots.txt
```
