---
title: "Template Context Reference"
description: "Reference for data, pages, page, and props in Veta templates and components."
---

# Template Context Reference

Veta templates receive a small root context:

```txt
data
pages
page
props
```

## `data`

Global data loaded from `data/`.

```html
{{ data.site.name }} {{ data.navigation.main }}
```

## `pages`

Array of all normalized pages.

```html
{% for item in pages %}
  <a href="{{ item.permalink }}">{{ item.title }}</a>
{% endfor %}
```

Each item includes the original page fields plus normalized fields such as `permalink`, `outputPath`, `template`, `generator`, and `index`.

## `page`

The current normalized page.

```html
<h1>{{ page.title }}</h1>
{{ page.content }}
```

For templated pages, `page.content` has already been processed through components and Markdown.

## `props`

Component props.

In page templates, `props` is usually empty.

In component templates, `props` contains tag attributes and `props.content`:

```html
<aside data-kind="{{ props.kind }}">
  {{ props.content }}
</aside>
```

## Template Helpers

Templates, includes, and components can call `load_data` to read local or remote data:

```html
{% set navigation = load_data("data/navigation.yaml")|parse_yaml %}
{% set site = load_data("data/site.json")|parse_json %}
```

See [Template Functions](../api/template-functions.md) for details.

Templates, includes, and components can also call `url` to generate current-page-relative links:

```html
<a href="{{ url(page.permalink) }}">{{ page.title }}</a>
<link rel="stylesheet" href="{{ url("/styles.css") }}">
```
