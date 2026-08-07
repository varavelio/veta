---
title: "JavaScript API"
description: "Understand where JavaScript runs in Veta and which context APIs are available."
---

# JavaScript API

Veta uses JavaScript for three kinds of project files:

```txt
data/*.js       -> global data producers
pages/*.js      -> page generators
filters/*.js    -> template filters
functions/*.js  -> template functions
```

JavaScript files are self-contained and synchronous. They do not use imports, module loading, or asynchronous promises. Each file must export one default function.

## Runtime Context

The default export receives a context object as its first argument.

Common context keys:

```txt
files
httpClient
parse
env
```

Additional context keys depend on where the file runs.

## `data/*.js`

Data files run while global data is being loaded, so they do not receive `data`.

```js
export default function({ env, files, httpClient }) {
  return {
    mode: env.VETA_MODE || "production",
  };
}
```

Return any JSON-compatible value. The value becomes part of `data` using the file path as its key.

Example:

```txt
data/github.js -> data.github
```

## `pages/*.js`

Page generators receive loaded global data:

```js
export default function({ data, parse }) {
  const { html } = parse.markdown("# Home");

  return [
    {
      permalink: "/",
      template: "base",
      title: data.site.name,
      content: html,
    },
  ];
}
```

Return an array of page objects. Templated content is passed unchanged and trusted to its template, while template-less content is written unchanged as raw output. Use `parse.markdown(text)` and `parse.renderComponents(text)` explicitly when a generator needs those transformations.

## `filters/*.js`

Filters receive the runtime context, the input value, and one optional parameter:

```js
export default function({ data }, input, parameter) {
  const prefix = parameter || data.site.name;
  return `${prefix}: ${input}`;
}
```

## `functions/*.js`

Template functions receive the runtime context followed by explicit template arguments:

```js
export default function({ page, data, files, parse }, value, length) {
  return String(value || page.title).slice(0, Number(length));
}
```

Use the file stem as a function in Pongo templates and components:

```html
{{ excerpt(page.content, 120) }}
```

Use it in a template:

```html
{{ page.title|prefix:"Post" }}
```

## No Global `Veta`

Veta does not expose runtime APIs through a global `Veta` object. Always use the context argument:

```js
export default function({ files }) {
  return files.listFiles("content/**/*.md");
}
```

## Console

The `console` object is available as a JavaScript global:

```js
export default function() {
  console.log("Generating pages");
  return [];
}
```

Supported methods are `debug`, `error`, `info`, `log`, and `warn`.

## Execution Model

Veta executes JavaScript synchronously. Promise-like return values are rejected.

Use synchronous calls only:

```js
export default function({ httpClient, parse }) {
  const response = httpClient.get("https://example.com/data.json");
  return parse.json(response.body);
}
```

Do not return a Promise:

```js
export default async function() {
  return [];
}
```

## API Pages

- [File API](./files.md)
- [HTTP Client](./http-client.md)
- [Parse API](./parse.md)
- [Environment And Console](./environment-and-console.md)
- [Frontmatter](./frontmatter.md)
