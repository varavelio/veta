---
title: "Parse API"
description: "Parse structured text, render Markdown, and explicitly resolve components from JavaScript."
---

# Parse API

The parse API is available as `parse` in JavaScript context objects. It parses structured text, renders Markdown bodies, and explicitly resolves component tags. File and HTTP APIs return text; call the required operations in the order your output needs.

```js
export default function({ files, parse }) {
  const { frontmatter, html } = parse.markdown(
    files.readFile("content/posts/hello.md"),
  );
  const content = parse.renderComponents(html);

  return [
    {
      permalink: "/posts/hello/",
      template: "post",
      title: frontmatter.title,
      content,
    },
  ];
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

Parses optional YAML or TOML frontmatter and renders the Markdown body to HTML.

```js
const post = parse.markdown(files.readFile("content/posts/hello.md"));
```

Return shape:

```js
{
  frontmatter: { title: "Hello" },
  content: "# Hello\n\nPost body.\n",
  html: "<h1>Hello</h1>\n<p>Post body.</p>\n"
}
```

- `frontmatter` is the parsed object.
- `content` is the raw body after frontmatter is removed.
- `html` is the Markdown-rendered body.

Without frontmatter, `frontmatter` is `{}`, `content` is the full input, and `html` is the full input rendered as Markdown.

## `parse.renderComponents(text)`

Resolves registered component tags in any supplied string and returns the transformed string:

```js
const content = parse.renderComponents(
  "<callout kind=\"warning\">Check the configuration.</callout>",
);
```

Only registered tags are resolved; other tags remain unchanged. Props, slot content, nested components, Pongo component context, includes, and inheritance work as they do elsewhere. This operation does not render Markdown.

Component-like text remains unchanged inside HTML attributes, comments, raw-text and code elements such as `script`, `style`, `code`, `pre`, `textarea`, and `title`, as well as Markdown inline code and fenced code blocks. Component template output is not scanned again, which keeps rendering a bounded, one-pass transformation. Excessively deep input or recursive calls from component template functions fail with a controlled render-limit error.

The caller controls ordering. For a Markdown file that may contain component tags, the recommended page-generator flow is:

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

When called by a page generator, component templates receive global `data`; `page` and `pages` do not exist yet because the generator is creating the page list. When a context-bound JavaScript template function calls it, available runtime `page` and `pages` values can flow into component rendering. Props come from each tag's attributes and slot content.

## Pongo Filter Distinction

The Pongo `parse_markdown` filter is unchanged: it returns `{ content, frontmatter }` and does not render Markdown. Use Pongo's separate `markdown` filter to render that `content`. JavaScript `parse.markdown(text)` returns the additional `html` field described above.

Parsed values are normalized into JavaScript-compatible values. Dates are exposed as strings.
