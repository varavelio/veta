---
title: "File API"
description: "Read project files, parse structured data, read Markdown frontmatter, list files, and create permalinks from JavaScript."
---

# File API

The file API is available as `files` in JavaScript context objects.

```js
export default function({ files }) {
  return files.listFiles("content/**/*.md");
}
```

All paths are relative to the project root. Absolute paths and `..` path traversal are rejected.

## `files.listFiles(pattern)`

Returns sorted project-relative file paths matching a glob pattern.

```js
const posts = files.listFiles("content/posts/**/*.md");
```

Use `files.listFiles(".")` to list every file in the project. Empty patterns are rejected.

## `files.readFile(path)`

Reads a file as a UTF-8 string.

```js
const robots = files.readFile("public/robots.txt");
```

## `files.readJsonFile(path)`

Reads and parses one JSON value.

```js
const site = files.readJsonFile("data/site.json");
```

JSON numbers are normalized into JavaScript-safe values. Multiple JSON values in one file are rejected.

## `files.readYamlFile(path)`

Reads and parses one YAML document.

```js
const navigation = files.readYamlFile("data/navigation.yaml");
```

Multiple YAML documents in one file are rejected.

## `files.readTomlFile(path)`

Reads and parses one TOML document.

```js
const theme = files.readTomlFile("data/theme.toml");
```

## `files.readMarkdownFile(path)`

Reads a Markdown file and parses optional YAML or TOML frontmatter.

```js
const post = files.readMarkdownFile("content/posts/hello.md");
```

Return shape:

```js
{
  content: "# Hello\n\nPost body.\n",
  frontmatter: { title: "Hello" },
  path: "content/posts/hello.md"
}
```

See [Frontmatter](./frontmatter.md) for details.

## `files.toPermalink(path, options)`

Converts a project-relative path into a pretty permalink.

```js
files.toPermalink("content/posts/hello.md", { stripPrefix: "content" });
// "/posts/hello/"
```

If the source file is an `index` file, the last segment is removed:

```js
files.toPermalink("content/docs/index.md", { stripPrefix: "content" });
// "/docs/"
```

Options:

```js
{
  stripPrefix: "content";
}
```

`stripPrefix` is optional. When present, Veta removes it as a complete path segment before generating the permalink. The source path must have that prefix.

## Security Rules

The file API rejects:

- empty file paths
- absolute paths
- Windows drive paths
- paths containing `..`
- symlink escapes outside the configured root

This keeps JavaScript file access confined to the project root.
