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
console
```

Additional context keys depend on where the file runs.

## `data/*.js`

Data files run while global data is being loaded, so they do not receive `data`.

```js
export default function({ env, files, httpClient, console }) {
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
export default function({ data, files, httpClient, env, console }) {
  return [
    {
      permalink: "/",
      template: "base",
      title: data.site.name,
      content: "# Home",
    },
  ];
}
```

Return an array of page objects.

## `filters/*.js`

Filters receive the runtime context, the input value, and one optional parameter:

```js
export default function({ data }, input, parameter) {
  const prefix = parameter || data.site.name;
  return `${prefix}: ${input}`;
}
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

The `console` object is available both globally and through context:

```js
export default function({ console }) {
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
