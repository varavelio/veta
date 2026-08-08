---
title: "Page Generators Reference"
description: "Complete contract for objects returned by JavaScript page generators."
icon: "file-code"
weight: 30
---

# Page Generators Reference

Page generators return arrays of page objects:

```js
export default function({ parse }) {
  const { html } = parse.markdown("# Home");

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
template: "pages/article.j2";
```

Do not prefix with `templates/`.

If omitted, the page is written as raw content.

## `content`

Required: no

Type: string

Defaults to an empty string.

For templated pages, content is passed unchanged and trusted to the selected template. Veta does not automatically render Markdown or resolve component tags. The generator must return the final format expected by the template, usually HTML. Use `parse.markdown(text)` and `parse.renderComponents(text)` explicitly when needed.

For template-less pages, content is written unchanged as raw output. It can contain HTML, Markdown, JSON, XML, text, or any other generated format.

A common content-file flow is:

```js
const { frontmatter, html } = parse.markdown(files.readFile(path));
const content = parse.renderComponents(html);

return {
  permalink: files.toPermalink(path, { stripPrefix: "content" }),
  template: "post",
  title: frontmatter.title,
  content,
};
```

## Extra Fields

Any extra fields are preserved and exposed as `page` in templates:

```js
{
  permalink: "/posts/hello/",
  template: "post",
  title: "Hello",
  date: "2026-06-30",
  tags: ["guide"],
  content: "<h1>Hello</h1>",
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
