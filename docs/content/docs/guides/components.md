---
title: "Components"
description: "Explicitly render reusable content components with props, slots, and Pongo templates."
icon: "blocks"
weight: 70
---

# Components

Components are reusable templates stored in `components/`. Veta discovers their custom tags, and JavaScript can resolve those tags explicitly with `parse.renderComponents(text)`. Page content is not scanned for components automatically.

## Basic Component

Create `components/note.html`:

```html
<aside class="note">
  {{ props.content }}
</aside>
```

Resolve it in a page generator:

```js
export default function({ parse }) {
  const { html } = parse.markdown(
    "Welcome to **Veta**.\n\n<note>Components are explicit.</note>",
  );
  const content = parse.renderComponents(html);

  return [
    {
      permalink: "/",
      template: "base",
      content,
    },
  ];
}
```

The component receives its slot as `props.content`. `parse.renderComponents` does not render Markdown; the example renders Markdown first and then resolves components. Calling it directly with `<note>Use **bold**.</note>` leaves the Markdown markers in the slot unchanged.

## Props

Attributes become string props:

```js
const content = parse.renderComponents(
  "<callout kind=\"warning\">Be careful.</callout>",
);
```

Component template:

```html
<aside data-kind="{{ props.kind }}">
  {{ props.content }}
</aside>
```

All attribute values are strings.

Component invocations use HTML-like syntax. Opening tags and quoted attributes
can span lines, and components without slot content can be self-closing:

```html
<callout
  kind="warning"
  title="Check the configuration"
>

This component has **Markdown slot content** when Markdown is rendered first.

</callout>

<status-badge
  label="Ready"
  tone="success"
/>
```

Keep attribute values quoted. A `>` inside a quoted value does not close the
tag.

## Component Names

Component tags are derived from file paths:

```txt
components/note.j2       -> <note>
components/ui/card.j2    -> <ui-card>
```

Valid component tags start with a lowercase letter and can contain lowercase letters, numbers, and hyphens. Double hyphens are rejected.

## Nested Components

Components can be nested in content:

```js
const content = parse.renderComponents(`
  <card title="Welcome">
    <note>Nested component content.</note>
  </card>
`);
```

The resolver handles registered nested tags present in the supplied source and leaves unregistered tags unchanged. It preserves component props and slots while recursively rendering nested components. Rendered component output is final and is not scanned again for more component tags, preventing templates from accidentally creating recursive expansion loops.

Component examples inside Markdown code spans or fences remain unchanged. The same applies to component-like text inside HTML attributes, comments, scripts, styles, code blocks, preformatted blocks, text areas, and titles.

## Explicit Ordering

The caller controls the transformation order. For Markdown files that may contain components, use:

```js
const { frontmatter, html } = parse.markdown(files.readFile(path));
const content = parse.renderComponents(html);
```

Markdown preserves multiline HTML-like component tags so the component pass can resolve both paired and self-closing forms. There is no implicit Markdown pass before or after component rendering. Do not pass component template output back through Markdown; the returned string can be assigned to a templated page as final trusted `content` or returned by a template-less page as raw output.

## Component Context

Component templates use the same root keys as Pongo templates when those values exist:

```txt
data
pages
page
props
```

`props` contains string attributes from the tag plus slot content in `props.content`.

When a page generator calls `parse.renderComponents`, global `data` is available to component templates. `page` and `pages` are not available yet because the generator is still creating the page list. If a context-bound JavaScript template function calls `parse.renderComponents`, its available runtime `page` and `pages` values flow into component rendering. Each resolved tag still supplies its own `props` and slot content.

## Component Inheritance

Components are Pongo templates, so they can use inheritance too:

```html
{# components/shell.j2 #}
<div class="shell {% block class %}{% endblock %}">
  {% block body %}{{ props.content }}{% endblock %}
</div>
```

```html
{# components/panel.j2 #}
{% extends "./shell.j2" %}

{% block class %}
  panel
{% endblock %}
```

Use relative paths with `./` or `../` inside component templates.

## Pongo Reuse

Component templates can include supporting templates or import macros through normal Pongo tags:

```html
{# components/note.html #}
<aside class="note">
  {% include "templates/brand.html" %}
  {{ props.content }}
</aside>
```

This is useful when the same markup or callable macro is needed from both page templates and content components. Supporting Pongo files can live anywhere under `templates/`; Veta does not prescribe their internal organization.

Explicit component resolution does not change Pongo behavior: component inheritance, includes, macro imports, relative paths, and the component template context continue to work normally.

## Ignored Component Files

Veta ignores component files or path segments that:

- start with `.`
- end with `~`
- end with `.tmp`

## Component Conflicts

If two files create the same tag, Veta picks the most specific deterministic winner and records the conflict internally. Avoid relying on conflicts. Use unique names.
