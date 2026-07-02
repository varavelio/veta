---
title: "File API"
description: "Read project files as text, list files, and create permalinks from JavaScript."
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
const site = parse.json(files.readFile("data/site.json"));
const post = parse.markdown(files.readFile("content/posts/hello.md"));
```

See [Parse API](./parse.md) and [Frontmatter](./frontmatter.md) for parsing structured content.

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
