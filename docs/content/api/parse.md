---
title: "Parse API"
description: "Parse text into structured data from JavaScript runtime context objects."
---

# Parse API

The parse API is available as `parse` in JavaScript context objects. It converts strings into structured values. File and HTTP APIs return text; use `parse` explicitly when you need data.

```js
export default function({ files, parse }) {
  const site = parse.json(files.readFile("data/site.json"));
  const navigation = parse.yaml(files.readFile("data/navigation.yaml"));
  const theme = parse.toml(files.readFile("data/theme.toml"));
  const post = parse.markdown(files.readFile("content/posts/hello.md"));

  return { site, navigation, theme, post };
}
```

## `parse.json(text)`

Parses one JSON value. Multiple JSON values are rejected.

```js
const site = parse.json("{\"title\":\"Veta\"}");
```

## `parse.yaml(text)`

Parses one YAML document. Multiple YAML documents are rejected.

```js
const navigation = parse.yaml("items:\n  - label: Docs\n");
```

## `parse.toml(text)`

Parses one TOML document.

```js
const theme = parse.toml("name = \"Clean\"\n");
```

## `parse.markdown(text)`

Parses optional YAML or TOML frontmatter without rendering Markdown to HTML.

```js
const post = parse.markdown(files.readFile("content/posts/hello.md"));
```

Return shape:

```js
{
  content: "# Hello\n\nPost body.\n",
  frontmatter: { title: "Hello" }
}
```

Use the `markdown` template filter when you want to render Markdown text to HTML.

Parsed values are normalized into JavaScript-compatible values. Dates are exposed as strings.
