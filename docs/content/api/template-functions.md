---
title: "Template Functions"
description: "Load local and remote data directly from Pongo templates with load_data."
---

# Template Functions

Veta registers template helpers for Pongo templates and components.

## `load_data`

`load_data` reads a local project file or a remote URL from a template, include, or component.

Use `load_data` inside native Pongo expressions. Assign values with Pongo's built-in `set` tag:

```html
{% set navigation = load_data("data/navigation.yaml") %} {% for item in
navigation.items %}
<a href="{{ item.href }}">{{ item.label }}</a>
{% endfor %}
```

The optional second argument is the format:

```html
{% set site = load_data("data/site.json") %} {% set readme =
load_data("content/readme.md", "text") %}

<h1>{{ site.title }}</h1>
{{ readme|markdown }}
```

### Local Files

Local paths are project-relative and can read files from the composed project and theme filesystem:

```html
{% set badge = load_data("data/badge.toml") %} {{ badge.label }}
```

Local paths must be relative. Absolute paths, Windows drive paths, empty paths, and paths containing `..` are rejected.

### Remote URLs

Remote URLs use HTTP `GET`:

```html
{% set repo = load_data("https://api.github.com/repos/varavelio/veta", "json")
%} {{ repo.stargazers_count }}
```

Only `http` and `https` URLs are allowed. Non-2xx responses fail the build.

You can set a timeout in milliseconds with the optional third argument:

```html
{% set value = load_data("https://example.com/data.json", "json", 5000) %}
```

### Formats

Supported formats are:

- `text`
- `json`
- `yaml`
- `toml`

When `format` is omitted, Veta detects it from the local file extension, remote `Content-Type`, or remote URL extension. Unknown extensions fall back to `text`.

```html
{% set message = load_data("content/message.txt", "text") %} {% set site =
load_data("data/site.json") %} {% set navigation =
load_data("data/navigation.yaml") %} {% set theme = load_data("data/theme.toml")
%}
```

Structured formats return normal template values:

```html
{{ site.title }} {{ navigation.items.0.label }} {{ theme.colors.primary }}
```

Text files return strings.
