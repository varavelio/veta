---
title: "Markdown"
description: "Render Markdown content, use frontmatter files, and understand Veta's Markdown pipeline."
---

# Markdown

Veta renders Markdown in templated page content and component slots. It uses GitHub Flavored Markdown features and allows inline HTML.

## Markdown In Page Content

When a page object has a `template`, its `content` is rendered as Markdown before the template is rendered:

```js
{
  permalink: "/about/",
  template: "base",
  title: "About",
  content: "# About\n\nThis is **Markdown**.",
}
```

Then the template can output the rendered HTML:

```html
<main>{{ page.content }}</main>
```

`page.content` is marked as trusted HTML after Veta renders it, so Pongo does not escape the generated HTML.

## Markdown In Component Slots

Component inner content is rendered through the same Markdown renderer:

```js
{
  permalink: "/",
  template: "base",
  content: "<note>Use **bold text** inside a component.</note>",
}
```

Component template:

```html
<aside>{{ props.content }}</aside>
```

## Markdown Files

Veta does not automatically discover Markdown pages. Use JavaScript generators to read files and create pages:

```js
export default function({ files }) {
  return files.listFiles("content/posts/**/*.md").map((path) => {
    const post = files.readMarkdownFile(path);

    return {
      permalink: files.toPermalink(path, { stripPrefix: "content" }),
      template: "post",
      title: post.frontmatter.title,
      content: post.content,
    };
  });
}
```

This keeps routing explicit and lets you decide how collections are sorted, filtered, paginated, or grouped.

## YAML Frontmatter

YAML frontmatter uses `---` delimiters:

```md
---
title: Hello World
draft: false
tags:
  - guide
  - intro
---

# Hello World

Post body.
```

## TOML Frontmatter

TOML frontmatter uses `+++` delimiters:

```md
+++
title = "Release Notes"
draft = false
tags = ["release", "notes"]

[meta]
author = "Veta"
+++

# Release Notes

Post body.
```

## `readMarkdownFile` Return Value

```js
const post = files.readMarkdownFile("content/posts/hello.md");
```

Returns:

```js
{
  content: "# Hello World\n\nPost body.\n",
  frontmatter: {
    title: "Hello World",
    draft: false,
    tags: ["guide", "intro"]
  },
  path: "content/posts/hello.md"
}
```

If a Markdown file has no frontmatter, `frontmatter` is an empty object and `content` is the full file content.

Frontmatter is detected only at the first line of the file. A `---` or `+++` line later in the document is treated as normal Markdown content.
