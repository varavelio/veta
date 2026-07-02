---
title: "Template Functions"
description: "Load local and remote text directly from Pongo templates with load_data."
---

# Template Functions

Veta registers template helpers for Pongo templates and components.

## `load_data`

`load_data` reads a local project file or a remote URL as text from a template, include, or component.

Use `load_data` inside native Pongo expressions. Assign values with Pongo's built-in `set` tag:

```html
{% set navigation = load_data("data/navigation.yaml")|parse_yaml %} {% for item
in navigation.items %}
<a href="{{ item.href }}">{{ item.label }}</a>
{% endfor %}
```

Without a parse filter, `load_data` returns a string:

```html
{% set readme = load_data("content/readme.md") %} {{ readme|markdown }}
```

### Local Files

Local paths are project-relative and can read files from the composed project and theme filesystem:

```html
{% set badge = load_data("data/badge.toml")|parse_toml %} {{ badge.label }}
```

Local paths must be relative. Absolute paths, Windows drive paths, empty paths, and paths containing `..` are rejected.

### Remote URLs

Remote URLs use HTTP `GET`:

```html
{% set repo =
load_data("https://api.github.com/repos/varavelio/veta")|parse_json %} {{
repo.stargazers_count }}
```

Only `http` and `https` URLs are allowed. Non-2xx responses fail the build.

### Parse Filters

Use parse filters to convert loaded text into structured values:

- `parse_json`
- `parse_yaml`
- `parse_toml`
- `parse_markdown`

```html
{% set message = load_data("content/message.txt") %} {% set site =
load_data("data/site.json")|parse_json %} {% set navigation =
load_data("data/navigation.yaml")|parse_yaml %} {% set theme =
load_data("data/theme.toml")|parse_toml %}
```

Parsed values return normal template values:

```html
{{ site.title }} {{ navigation.items.0.label }} {{ theme.colors.primary }}
```
