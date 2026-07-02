---
title: "Markdown Frontmatter"
description: "Parse YAML and TOML frontmatter with parse.markdown and parse_markdown."
---

# Markdown Frontmatter

`parse.markdown(text)` in JavaScript and `parse_markdown` in Pongo templates support optional frontmatter at the start of a Markdown string.

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
  content: "# Hello\n\nBody.\n",
  frontmatter: {
    title: "Hello",
    draft: false,
    tags: ["guide", "intro"]
  }
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
  frontmatter: {}
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
