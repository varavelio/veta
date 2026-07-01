---
title: "Troubleshooting"
description: "Common Veta errors and how to fix them."
---

# Troubleshooting

This page explains common errors and the shortest path to fixing them.

## Config File Not Found

Veta looks for:

```txt
veta.yaml
veta.yml
.veta.yaml
.veta.yml
```

It searches from the current directory upward.

Fixes:

```sh
veta init
veta build --config ./veta.yaml
veta dev --config ./veta.yaml
```

## Invalid Page Object

Every page must be an object with a string `permalink`.

Valid:

```js
{
  permalink: "/about/",
  template: "base",
  content: "About"
}
```

If you see an error about `layout`, rename it to `template`.

## Template Not Found

Page templates are relative to `templates/`:

```js
template: "base";
```

Do not write:

```js
template: "templates/base.html";
```

If extensionless lookup is ambiguous, include the extension:

```js
template: "base.pongo";
```

## Duplicate Output Path

Two pages cannot generate the same output file.

These conflict:

```js
{
  permalink: "/about/";
}
{
  permalink: "/about/index.html";
}
```

Change one permalink.

## Tailwind Input Missing

If `tailwindcss.stylesheet: styles.css` is set, Veta expects:

```txt
public/styles.css
```

Create the file or remove `tailwindcss.stylesheet` to disable Tailwind CSS.

## Public Asset Collision

Generated files and public files share the same output namespace.

These conflict:

```txt
page permalink: /robots.txt
public/robots.txt
```

Move one of them.

## JavaScript Promise Error

Veta JavaScript is synchronous. Do not use `async` default exports or return promises.

Use synchronous `httpClient` calls instead:

```js
export default function({ httpClient }) {
  const response = httpClient.get("https://example.com/data.json");
  return JSON.parse(response.body);
}
```

## Frontmatter Error

Frontmatter must start on the first line and close with the same delimiter:

```md
---
title: Hello
---

# Hello
```

Use `---` for YAML and `+++` for TOML.
