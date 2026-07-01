---
title: "Markdown Frontmatter"
description: "Parse YAML and TOML frontmatter with files.readMarkdownFile."
---

# Markdown Frontmatter

`files.readMarkdownFile(path)` supports optional frontmatter at the start of a Markdown file.

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
const post = files.readMarkdownFile("content/posts/hello.md");
```

```js
{
  content: "# Hello\n\nBody.\n",
  frontmatter: {
    title: "Hello",
    draft: false,
    tags: ["guide", "intro"]
  },
  path: "content/posts/hello.md"
}
```

One blank line immediately after the closing delimiter is removed from `content`.

## Files Without Frontmatter

```md
# Plain Markdown

No frontmatter.
```

Returns:

```js
{
  content: "# Plain Markdown\n\nNo frontmatter.\n",
  frontmatter: {},
  path: "content/plain.md"
}
```

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
