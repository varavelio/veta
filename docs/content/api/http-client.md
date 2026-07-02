---
title: "HTTP Client"
description: "Fetch remote data synchronously from Veta JavaScript files."
---

# HTTP Client

The HTTP client is available as `httpClient` in JavaScript context objects.

It is synchronous and supports HTTP and HTTPS URLs only.

## Shortcut Methods

```js
httpClient.get(url, options);
httpClient.post(url, options);
httpClient.put(url, options);
httpClient.patch(url, options);
httpClient.delete(url, options);
httpClient.head(url, options);
```

Example:

```js
export default function({ httpClient, parse }) {
  const response = httpClient.get(
    "https://api.github.com/repos/varavelio/veta",
    {
      headers: {
        Accept: "application/vnd.github+json",
      },
    },
  );

  if (!response.ok) {
    throw new Error(`GitHub returned ${response.status}`);
  }

  return parse.json(response.body);
}
```

## Explicit Request Method

```js
httpClient.request("GET", "https://example.com/data.json");
```

The method is trimmed and uppercased. Empty methods or methods containing whitespace are rejected.

## Options

```js
{
  headers: {
    "Accept": "application/json",
    "X-Trace": ["one", "two"]
  },
  body: "raw body",
  timeoutMs: 5000
}
```

`body` must be a string.

For JSON request bodies, use `JSON.stringify` and set the content type yourself:

```js
httpClient.post("https://example.com/api", {
  body: JSON.stringify({ message: "hello" }),
  headers: { "Content-Type": "application/json" },
});
```

`timeoutMs` must be a positive number. The default timeout is 30 seconds.

## Response Shape

```js
{
  body: "...",
  headers: {
    "Content-Type": ["application/json"]
  },
  ok: true,
  status: 200,
  statusText: "OK",
  url: "https://example.com/data.json"
}
```

`ok` is `true` for status codes from 200 through 299.

`body` is always a string. Use `parse.json`, `parse.yaml`, or another parser when you need structured data.

## Development Advice

`veta dev` performs a full rebuild on file changes. If a data script fetches slow remote APIs, it will fetch them again on rebuild. For development, consider using `env` to return local mock data.
