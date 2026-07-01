---
title: "Page Generators Reference"
description: "Complete contract for objects returned by JavaScript page generators."
---

# Page Generators Reference

Page generators return arrays of page objects:

```js
export default function() {
  return [
    {
      permalink: "/",
      template: "base",
      title: "Home",
      content: "# Home",
    },
  ];
}
```

## `permalink`

Required: yes

Type: string

The public URL path for the generated page.

Examples:

```txt
/
/about/
/feed.xml
/llms.txt
```

## `template`

Required: no

Type: string

Template name relative to `templates/`.

Examples:

```js
template: "base";
template: "pages/article.pongo";
```

Do not prefix with `templates/`.

If omitted, the page is written as raw content.

## `content`

Required: no

Type: string

Defaults to an empty string.

For templated pages, content is processed through components and Markdown before the template is rendered.

For raw pages, content is written directly.

## Extra Fields

Any extra fields are preserved and exposed as `page` in templates:

```js
{
  permalink: "/posts/hello/",
  template: "post",
  title: "Hello",
  date: "2026-06-30",
  tags: ["guide"],
  content: "# Hello",
}
```

Template:

```html
<time>{{ page.date }}</time>
```

## Normalized Fields

Veta also exposes normalized fields on `page`:

```txt
content
generator
index
outputPath
permalink
template
```

`generator` is the page generator file path.

`index` is the page's index in that generator's returned array.

`outputPath` is the generated file path inside the output directory.

## Removed Field: `layout`

`layout` is rejected. Use `template` instead.
