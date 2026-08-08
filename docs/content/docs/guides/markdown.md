---
title: "Markdown"
description: "Explicitly render Markdown, parse frontmatter, and combine Markdown with components."
icon: "file-text"
weight: 50
---

# Markdown

Veta provides explicit Markdown parsing through JavaScript and Pongo filters. It uses GitHub Flavored Markdown features and allows inline HTML. Page content is never rendered as Markdown automatically.

## Markdown In Page Content

Call `parse.markdown(text)` in a page generator and pass its `html` result to the page:

```js
export default function({ parse }) {
  const { html } = parse.markdown("# About\n\nThis is **Markdown**.");

  return [
    {
      permalink: "/about/",
      template: "base",
      title: "About",
      content: html,
    },
  ];
}
```

The template outputs that HTML:

```html
<main>{{ page.content }}</main>
```

Templated `page.content` is passed unchanged and trusted to the selected template, so Pongo does not escape it. The generator is responsible for producing the expected final format. Template-less content is also unchanged and is written as raw output, which makes raw Markdown pages possible.

## Markdown And Components

Markdown rendering and component resolution are independent operations. Call them explicitly in the order required by the source. The recommended generator flow is:

```js
const { frontmatter, html } = parse.markdown(files.readFile(path));
const content = parse.renderComponents(html);
return content;
```

`parse.renderComponents(text)` resolves registered component tags but does not render Markdown. For example:

```js
export default function({ files, parse }) {
  const path = "content/about.md";
  const { frontmatter, html } = parse.markdown(files.readFile(path));
  const content = parse.renderComponents(html);

  return [
    {
      permalink: "/about/",
      template: "base",
      title: frontmatter.title,
      content,
    },
  ];
}
```

Component slot content is not given an additional Markdown pass. See [Components](/docs/guides/components/) for component behavior and context.

Raw HTML and HTML-like component tags are preserved, including opening tags whose quoted attributes span multiple lines. Both paired tags and self-closing tags therefore remain available to the following component pass. Markdown and HTML files are trusted author input in Veta; this rendering step does not sanitize scripts, event attributes, or other raw HTML.

## Markdown Files

Veta does not automatically discover Markdown pages. Use JavaScript generators to read files and create pages:

```js
export default function({ files, parse }) {
  return files.listFiles("content/posts/**/*.md").map((path) => {
    const { frontmatter, html } = parse.markdown(files.readFile(path));
    const content = parse.renderComponents(html);

    return {
      permalink: files.toPermalink(path, { stripPrefix: "content" }),
      template: "post",
      title: frontmatter.title,
      content,
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

## `parse.markdown` Return Value

```js
const post = parse.markdown(files.readFile("content/posts/hello.md"));
```

Returns:

```js
{
  frontmatter: {
    title: "Hello World",
    draft: false,
    tags: ["guide", "intro"]
  },
  content: "# Hello World\n\nPost body.\n",
  html: "<h1>Hello World</h1>\n<p>Post body.</p>\n"
}
```

`content` is the raw Markdown body and `html` is that body rendered as Markdown. If a Markdown file has no frontmatter, `frontmatter` is an empty object and `content` is the full input.

Frontmatter is detected only at the first line of the file. A `---` or `+++` line later in the document is treated as normal Markdown content.

## Pongo Markdown Filters

The Pongo `parse_markdown` filter keeps its template-specific return shape, `{ content, frontmatter }`, and does not render Markdown. Pipe `content` through the separate `markdown` filter when HTML is required:

```html
{% set post = load_data("content/post.md")|parse_markdown %}
{{ post.content|markdown }}
```

This differs from JavaScript `parse.markdown(text)`, which also returns `html`.
