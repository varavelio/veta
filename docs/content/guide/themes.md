---
title: "Themes"
description: "Compose local or remote Veta themes with project files and project-level overrides."
---

# Themes

Themes let you share templates, components, filters, data, and public assets across projects.

Configure a theme with `theme.source`:

```yaml
theme:
  source: "./themes/clean"
```

## What Themes Can Provide

A theme can contain these top-level directories:

```txt
templates/
components/
filters/
data/
public/
```

Other top-level directories are ignored by the theme overlay.

## Project Files Override Theme Files

Veta composes the theme and project into one filesystem. Project files win over theme files.

Example:

```txt
theme/templates/base.j2
templates/base.j2
```

The project's `templates/base.j2` overrides the theme template.

This lets a project use most of a theme while customizing selected files.

## Local Themes

Use a relative path:

```yaml
theme:
  source: "./themes/blog"
```

The path is resolved from the project root.

## GitHub Themes

Remote theme sources use a GitHub-style reference:

```yaml
theme:
  source: "owner/repository@ref"
```

Use tags or commit references when you want reproducible builds.

Veta caches remote themes under its runtime cache directory.

## Pages Stay In The Project

Themes provide building blocks. Projects still declare the pages they want to output through `pages/*.js`.

This keeps site structure explicit and prevents a theme from unexpectedly creating routes.

## Theme Data

Themes can provide data files, but project data can override them. A common pattern is:

```txt
theme/data/theme.json
data/theme.json
```

The project file can customize names, colors, navigation, or other theme-facing values.

## Theme Configuration Defaults

When building reusable themes, prefer exposing user-configurable defaults through `data/site_defaults.yaml` in the theme. Projects can then override only the values they care about with `data/site.yaml`:

```txt
theme/data/site_defaults.yaml
data/site.yaml
```

This keeps the theme defaults and project overrides available as separate template values:

```txt
data.site_defaults
data.site
```

Example theme defaults:

```yaml
# theme/data/site_defaults.yaml
name: "Clean Theme"
description: "A clean Veta site."

brand:
  color: "blue"
  logo: "/images/logo.svg"
```

Example project overrides:

```yaml
# data/site.yaml
name: "My Site"

brand:
  color: "purple"
```

Then theme templates can prefer project values and fall back to theme defaults:

```html
{% if data.site and data.site.name %}
  {{ data.site.name }}
{% else %}
  {{ data.site_defaults.name }}
{% endif %}
```

Use this pattern when you want partial project customization. If a theme provides `data/site.yaml` and the project also provides `data/site.yaml`, the project file replaces the theme file completely.

Veta does not deep-merge data files automatically. Keep fallbacks explicit in templates so theme behavior stays easy to understand.
