---
title: "Filters"
description: "Use built-in template filters and add custom JavaScript filters."
---

# Filters

Filters transform values inside templates. Veta includes built-in filters and can load custom JavaScript filters from `filters/`.

## Built-In Filters

### `json`

Serializes a value as JSON:

```html
<script type="application/json">
  {{ data.site|json }}
</script>
```

### `markdown`

Renders a string as Markdown:

```html
{{ page.summary | markdown }}
```

The output is trusted HTML.

### Parse Filters

Parse filters convert strings into structured template values:

```html
{% set site = load_data("data/site.json") | parse_json %} {% set navigation =
load_data("data/navigation.yaml") | parse_yaml %} {% set theme =
load_data("data/theme.toml") | parse_toml %} {% set post =
load_data("content/post.md") | parse_markdown %}
```

`parse_markdown` parses YAML or TOML frontmatter and returns `{ content, frontmatter }`. It does not render Markdown to HTML; use `markdown` for rendering.

## Custom JavaScript Filters

Create `filters/titlecase.js`:

```js
export default function({ data }, input) {
  return String(input)
    .split(" ")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}
```

Use it in a template:

```html
<h1>{{ page.title|titlecase }}</h1>
```

The filter file name becomes the filter name. `filters/titlecase.js` becomes `titlecase`.

## Filter Parameters

Filters can receive one parameter:

```html
{{ page.title|prefix:"Post: " }}
```

```js
export default function(runtime, input, parameter) {
  return `${parameter}${input}`;
}
```

When a filter is called without a parameter, the third argument is `null`-like from the JavaScript side.

## Runtime Context

Custom filters receive the JavaScript runtime context as the first argument:

```js
export default function({ data, env }, input, parameter) {
  return `${data.site.name}: ${input}`;
}
```

Filters are synchronous. Promises are not supported.

## Directory Rules

`filters/` is flat. Nested filter directories are not supported.

Every filter file must end in `.js`.
