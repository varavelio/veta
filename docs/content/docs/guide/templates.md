---
title: "Templates"
description: "Use Pongo templates, inheritance, includes, macros, filters, and the Veta template context."
---

# Templates

Pongo page templates and supporting files live in `templates/`. Veta does not prescribe subdirectories inside it, so each project can organize layouts, fragments, and macro libraries as needed. A page object uses a template by setting `template`:

```js
export default function({ parse }) {
  const { html } = parse.markdown("# Welcome");

  return [
    {
      permalink: "/",
      template: "base",
      title: "Home",
      content: html,
    },
  ];
}
```

Veta resolves the name relative to `templates/`. It passes the generator's `content` string unchanged and trusted to the template; Markdown rendering and component resolution are explicit generator operations.

## Template Names

Veta supports any template extension, but `.j2` is the recommended convention for Pongo templates and components. Pongo uses Jinja-style syntax, and many editors and formatters already recognize `.j2` files well.

```txt
templates/base.j2
templates/navigation.j2
components/card.j2
```

You can include the file extension:

```js
template: "base.j2";
```

Or omit it:

```js
template: "base";
```

When the extension is omitted, Veta scans for a non-ignored file with the same stem. For example, `base` can resolve to `templates/base.j2`.

If more than one file matches the same extensionless name, Veta reports an ambiguous template error.

## Template Context

Templates receive exactly these root keys:

```txt
data
pages
page
props
```

Example:

```html
<title>{{ page.title }} - {{ data.site.name }}</title>

{% for item in pages %}
  <a href="{{ item.permalink }}">{{ item.title }}</a>
{% endfor %}

<main>{{ page.content }}</main>
```

`props` is usually empty in page templates. It is populated when rendering components.

## Inheritance

Pongo inheritance works inside `templates/`:

```html
{# templates/base.j2 #}
<!DOCTYPE html>
<html lang="en">
  <head>
    <title>{% block title %}{{ data.site.name }}{% endblock %}</title>
  </head>
  <body>
    {% block main %}{% endblock %}
  </body>
</html>
```

```html
{# templates/pages/article.j2 #}
{% extends "../base.j2" %}

{% block title %}
  {{ page.title }} | {{ block.Super }}
{% endblock %}

{% block main %}
  <article>{{ page.content }}</article>
{% endblock %}
```

Use `./` or `../` for relative paths in `extends` and `include` statements.

## Includes

Pongo templates can include other files by project-relative path:

```html
{% include "templates/brand.html" %}
```

Includes receive the current template context, including `data`, `pages`, `page`, and `props`.

Use `with` to provide values explicitly and `only` to isolate an included file from the current context:

```html
{% include "templates/user-card.j2" with user=page.author only %}
```

Page templates and components use the same loader, so both can reuse files under `templates/`.

## Macros And Imports

Macros define callable template fragments. Add `export` when a macro must be imported from another file:

```html
{# templates/ui.j2 #}
{% macro button(text, href, tone="primary") export %}
  <a class="button button-{{ tone }}" href="{{ href }}">{{ text }}</a>
{% endmacro %}
```

Import the exported names that the caller needs. Imports can use aliases:

```html
{% import "templates/ui.j2" button as action %}
{{ action("Read the guide", "/guide/") }}
```

Macros can also be defined and called in the same file without `export`. Macro files use the normal template loader, including extensionless names and project-over-theme overrides.

## Loading Data

Pongo templates and components can load local or remote data with `load_data`:

```html
{% set navigation = load_data("data/navigation.yaml")|parse_yaml %}
{% set site = load_data("data/site.json")|parse_json %}
```

Use `load_data` for template-specific data. Use global `data/` files for data shared across the whole site. See [Template Functions](../api/template-functions.md) for the full API.

## Functions

Pongo templates and components can call built-in functions such as `url`, `regex_replace`, and `load_data`. Projects can add custom JavaScript functions in `functions/`:

```js
// functions/excerpt.js
export default function({ page }, value, length) {
  return String(value || page.title).slice(0, Number(length));
}
```

```html
{{ excerpt(page.content, 120) }}
```

See [Template Functions](../api/template-functions.md) for details.

## Filters

Veta registers built-in filters and custom filters:

```html
<script type="application/json">
  {{ page|json }}
</script>

{{ page.summary|markdown }}
```

Custom JavaScript filters live in `filters/` and are documented in [Filters](./filters.md).

## Ignored Template Files

Veta ignores template files or path segments that:

- start with `.`
- end with `~`
- end with `.tmp`

This lets editors keep temporary files in the project without affecting builds.
