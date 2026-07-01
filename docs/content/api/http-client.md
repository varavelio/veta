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
export default function({ httpClient }) {
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

  return JSON.parse(response.body);
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
  json: { message: "hello" },
  timeoutMs: 5000
}
```

`body` must be a string.

`json` is serialized to JSON and sets `Content-Type: application/json` when the header is not already provided.

Use either `body` or `json`, not both.

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

## Development Advice

`veta dev` performs a full rebuild on file changes. If a data script fetches slow remote APIs, it will fetch them again on rebuild. For development, consider using `env` to return local mock data.
