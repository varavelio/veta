---
title: "Components"
description: "Create reusable content components with props, slots, Markdown, and Pongo templates."
---

# Components

Components are reusable templates stored in `components/`. Veta discovers them and lets you use them as custom tags inside page content.

## Basic Component

Create `components/note.html`:

```html
<aside class="note">
  {{ props.content }}
</aside>
```

Use it in page content:

```js
{
  permalink: "/",
  template: "base",
  content: "<note>This supports **Markdown**.</note>",
}
```

The component receives the rendered slot as `props.content`.

## Props

Attributes become string props:

```js
content: "<callout kind=\"warning\">Be careful.</callout>";
```

Component template:

```html
<aside data-kind="{{ props.kind }}">
  {{ props.content }}
</aside>
```

All attribute values are strings.

## Component Names

Component tags are derived from file paths:

```txt
components/note.html        -> <note>
components/ui/card.pongo    -> <ui-card>
```

Valid component tags start with a lowercase letter and can contain lowercase letters, numbers, and hyphens. Double hyphens are rejected.

## Nested Components

Components can be nested in content:

```html
<card title="Welcome">
  <note>Nested **Markdown** content.</note>
</card>
```

Veta parses registered component tags and renders each component with the current page context.

## Component Context

Component templates receive:

```txt
data
pages
page
props
```

`props` contains attributes plus `props.content`.

## Component Inheritance

Components are Pongo templates, so they can use inheritance too:

```html
{# components/shell.pongo #}
<div class="shell {% block class %}{% endblock %}">
  {% block body %}{{ props.content }}{% endblock %}
</div>
```

```html
{# components/panel.pongo #} {% extends "./shell.pongo" %} {% block class
%}panel{% endblock %}
```

Use relative paths with `./` or `../` inside component templates.

## Ignored Component Files

Veta ignores component files or path segments that:

- start with `.`
- end with `~`
- end with `.tmp`

## Component Conflicts

If two files create the same tag, Veta picks the most specific deterministic winner and records the conflict internally. Avoid relying on conflicts. Use unique names.
