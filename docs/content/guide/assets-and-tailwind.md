---
title: "Assets And Tailwind CSS"
description: "Use public assets and Veta's embedded Tailwind CSS standalone integration."
---

# Assets And Tailwind CSS

Static assets live in `public/`. Tailwind CSS is configured through `veta.yaml` and uses a stylesheet inside `public/` as its input.

## Public Assets

Files in `public/` are copied to the output root:

```txt
public/robots.txt           -> dist/robots.txt
public/images/logo.svg      -> dist/images/logo.svg
```

Public files are copied as-is. Veta does not minify or transform copied public assets.

## Tailwind CSS Input

The starter uses `public/styles.css`:

```css
@import "tailwindcss";
```

Configure it with:

```yaml
tailwindcss:
  stylesheet: styles.css
  minify: true
```

`stylesheet` is relative to `public/`, so `styles.css` means `public/styles.css`.

## Generated CSS Output

Veta writes the compiled stylesheet to the build output using the same path:

```txt
public/styles.css           -> dist/styles.css
```

Tailwind scans the materialized output directory, so classes used in generated HTML are included.

## Minification

`tailwindcss.minify: true` minifies the generated stylesheet through Tailwind CSS.

This setting is separate from `html.minify`, which only affects generated `.html` files.

## Disabling Tailwind CSS

Remove `tailwindcss.stylesheet` or leave it blank:

```yaml
tailwindcss:
  minify: true
```

Without `stylesheet`, Veta does not run Tailwind CSS.

## Practical Pattern

Use `public/styles.css` for your Tailwind entrypoint and keep images, fonts, and static files under `public/`:

```txt
public/
  styles.css
  fonts/inter.woff2
  images/logo.svg
  robots.txt
```
