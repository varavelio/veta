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

The complete `pages` list exists after page generators have returned. It is not available while a generator is still creating that list.

## `page`

The current normalized page.

```html
<h1>{{ page.title }}</h1>
{{ page.content }}
```

For templated pages, `page.content` is the generator's unchanged, trusted string. Veta does not automatically render Markdown or resolve components before template rendering.

## `props`

Component props.

In page templates, `props` is usually empty.

In component templates, `props` contains tag attributes and `props.content`:

```html
<aside data-kind="{{ props.kind }}">
  {{ props.content }}
</aside>
```

Component context depends on where `parse.renderComponents(text)` is called. Page generators provide global `data`, but not `page` or `pages` because those pages are still being created. A context-bound JavaScript template function can pass its available runtime `page` and `pages` values into component rendering. Tag attributes and slot content always supply the rendered component's `props`.

## Template Helpers

Templates, includes, and components can call built-in and custom template functions. `load_data` reads local or remote data:

```html
{% set navigation = load_data("data/navigation.yaml")|parse_yaml %}
{% set site = load_data("data/site.json")|parse_json %}
```

See [Template Functions](../api/template-functions.md) for details.

They can also call `url` to generate current-page-relative links:

```html
<a href="{{ url(page.permalink) }}">{{ page.title }}</a>
<link rel="stylesheet" href="{{ url("/styles.css") }}">
```

Custom functions from `functions/*.js` are available by file stem:

```html
{{ excerpt(page.content, 120) }}
```
