---
title: "Markdown Frontmatter"
description: "Parse YAML and TOML frontmatter with parse.markdown and parse_markdown."
icon: "align-start-vertical"
weight: 70
---

# Markdown Frontmatter

`parse.markdown(text)` in JavaScript and `parse_markdown` in Pongo templates support optional frontmatter at the start of a Markdown string. Their return shapes differ: JavaScript also renders the body into an `html` field, while the Pongo filter keeps `{ content, frontmatter }`.

Supported delimiters:

```txt
---   YAML
+++   TOML
```

Frontmatter is detected only when the first line is exactly `---` or `+++`.

## YAML Frontmatter

```md
---
title: Hello
draft: false
tags:
  - guide
  - intro
---

# Hello

Body.
```

## TOML Frontmatter

```md
+++
title = "Hello"
draft = false
tags = ["guide", "intro"]

[meta]
author = "Veta"
+++

# Hello

Body.
```

## Return Shape

```js
const post = parse.markdown(files.readFile("content/posts/hello.md"));
```

```js
{
  frontmatter: { title: "Hello", draft: false, tags: ["guide", "intro"] },
  content: "# Hello\n\nBody.\n",
  html: "<h1>Hello</h1>\n<p>Body.</p>\n"
}
```

`content` is the raw body, and `html` is the Markdown-rendered body. One blank line immediately after the closing delimiter is removed from `content` before `html` is rendered.

## Files Without Frontmatter

```md
# Plain Markdown

No frontmatter.
```

Returns:

```js
{
  frontmatter: {},
  content: "# Plain Markdown\n\nNo frontmatter.\n",
  html: "<h1>Plain Markdown</h1>\n<p>No frontmatter.</p>\n"
}
```

Without frontmatter, `content` is the full input.

## Pongo `parse_markdown`

The Pongo filter remains a frontmatter parser only:

```html
{% set post = load_data("content/posts/hello.md")|parse_markdown %}
{{ post.content|markdown }}
```

It returns `{ content, frontmatter }`; use the separate `markdown` filter to produce HTML.

## Validation

Veta rejects:

- missing closing delimiters
- malformed YAML
- malformed TOML
- frontmatter that does not parse to an object
- multiple YAML documents
- non-finite numbers such as `NaN` or `Inf`
- maps with non-string keys

Parsed values are normalized into JavaScript-compatible values. Dates are exposed as strings.
