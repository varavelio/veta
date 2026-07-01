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

You can include the file extension:

```js
template: "base.html";
```

Or omit it:

```js
template: "base";
```

When the extension is omitted, Veta scans for a non-ignored file with the same stem. For example, `base` can resolve to `templates/base.html` or `templates/base.pongo`.

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
{# templates/base.pongo #}
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
{# templates/pages/article.pongo #} {% extends "../base.pongo" %} {% block title
%}{{ page.title }} | {{ block.Super }}{% endblock %} {% block main %}
<article>{{ page.content }}</article>
{% endblock %}
```

Use `./` or `../` for relative paths in `extends` and `include` statements.

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
