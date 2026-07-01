---
title: "Template Functions"
description: "Load local and remote data directly from Pongo templates with load_data."
---

# Template Functions

Veta registers template helpers for Pongo templates and components.

## `load_data`

`load_data` reads a local project file or a remote URL from a template, include, or component.

Use the tag form when you want named options:

```html
{% load_data path="data/navigation.yaml" as navigation %} {% for item in
navigation.items %}
<a href="{{ item.href }}">{{ item.label }}</a>
{% endfor %}
```

Use the function form inside expressions. Pongo function calls use positional arguments, so the optional second argument is the format:

```html
{% set site = load_data("data/site.json") %} {% set readme =
load_data("content/readme.md", "text") %}

<h1>{{ site.title }}</h1>
{{ readme|markdown }}
```

### Local Files

Local paths are project-relative and can read files from the composed project and theme filesystem:

```html
{% load_data path="data/badge.toml" as badge %} {{ badge.label }}
```

Local paths must be relative. Absolute paths, Windows drive paths, empty paths, and paths containing `..` are rejected.

### Remote URLs

Remote URLs use HTTP `GET`:

```html
{% load_data url="https://api.github.com/repos/varavelio/veta" format="json" as
repo %} {{ repo.stargazers_count }}
```

Only `http` and `https` URLs are allowed. Non-2xx responses fail the build.

You can set a timeout in milliseconds on the tag form:

```html
{% load_data url="https://example.com/data.json" timeout_ms=5000 as value %}
```

### Formats

Supported formats are:

- `text`
- `json`
- `yaml`
- `toml`

When `format` is omitted, Veta detects it from the local file extension, remote `Content-Type`, or remote URL extension. Unknown extensions fall back to `text`.

```html
{% load_data path="content/message.txt" format="text" as message %} {% load_data
path="data/site.json" as site %} {% load_data path="data/navigation.yaml" as
navigation %} {% load_data path="data/theme.toml" as theme %}
```

Structured formats return normal template values:

```html
{{ site.title }} {{ navigation.items.0.label }} {{ theme.colors.primary }}
```

Text files return strings.
