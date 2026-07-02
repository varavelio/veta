---
title: "Templates"
description: "Use Pongo templates, template inheritance, filters, and the Veta template context."
---

# Templates

Templates live in `templates/` and are rendered with Pongo. A page object uses a template by setting `template`:

```js
{
  permalink: "/",
  template: "base",
  title: "Home",
  content: "# Welcome",
}
```

Veta resolves the name relative to `templates/`.

## Template Names

Veta supports any template extension, but `.j2` is the recommended convention for templates, includes, and components. Pongo uses Jinja-style syntax, and many editors and formatters already recognize `.j2` files well.

```txt
templates/base.j2
includes/nav.j2
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
{# templates/pages/article.j2 #} {% extends "../base.j2" %} {% block title
%}{{ page.title }} | {{ block.Super }}{% endblock %} {% block main %}
<article>{{ page.content }}</article>
{% endblock %}
```

Use `./` or `../` for relative paths in `extends` and `include` statements.

## Includes

Shared Pongo fragments live in `includes/`. Templates can include them by project-relative path:

```html
{% include "includes/brand.html" %}
```

Includes receive the current template context, including `data`, `pages`, `page`, and `props`.

Use `includes/` for reusable markup shared between templates and components, such as buttons, badges, tables, and navigation fragments. Markup used only by page templates can stay in `templates/`. Use `components/` when you need a custom tag inside page content.

Pongo can include files from other project directories, but `includes/` is Veta's standard convention and is watched by `veta dev` by default.

## Loading Data

Templates, includes, and components can load local or remote data with `load_data`:

```html
{% set navigation = load_data("data/navigation.yaml")|parse_yaml %} {% set site
= load_data("data/site.json")|parse_json %}
```

Use `load_data` for template-specific data. Use global `data/` files for data shared across the whole site. See [Template Functions](../api/template-functions.md) for the full API.

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
