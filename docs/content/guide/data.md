---
title: "Data"
description: "Load global JSON, YAML, TOML, and JavaScript data into Veta templates and generators."
---

# Data

Global data lives in `data/`. Veta loads data before page generation and exposes it as `data` in JavaScript generators, templates, components, and filters.

## Supported Formats

Veta supports:

```txt
.json
.yaml
.yml
.toml
.js
```

Examples:

```txt
data/site.json
data/navigation.yaml
data/theme.toml
data/github.js
```

## Data Keys

Data keys come from file paths without extensions:

```txt
data/site.json              -> data.site
data/navigation.yaml        -> data.navigation
data/theme/colors.toml      -> data.theme.colors
```

Data file stems must be valid JavaScript-style identifiers. Prefer names like `site.json`, `navigation.yaml`, and `theme/colors.toml`. Avoid names like `site-name.json` because hyphens do not produce ergonomic template keys.

## JSON Data

```json
{
  "name": "Veta Docs",
  "description": "Documentation built with Veta."
}
```

Use it in a template:

```html
<title>{{ data.site.name }}</title>
```

## YAML Data

```yaml
main:
  - label: Home
    href: /
  - label: Docs
    href: /docs/
```

Use it in a template:

```html
{% for item in data.navigation.main %}
<a href="{{ item.href }}">{{ item.label }}</a>
{% endfor %}
```

YAML data files support one YAML document. Multiple YAML documents in one file are rejected.

## TOML Data

```toml
name = "Clean"

[colors]
primary = "blue"
```

Use it in a template:

```html
<p>{{ data.theme.colors.primary }}</p>
```

## JavaScript Data

JavaScript data files export a default function and return a value:

```js
export default function({ env, httpClient }) {
  if (env.VETA_MODE === "development") {
    return { stars: 0, repo: "local/mock" };
  }

  const response = httpClient.get(
    "https://api.github.com/repos/varavelio/veta",
  );
  const repo = JSON.parse(response.body);

  return {
    repo: repo.full_name,
    stars: repo.stargazers_count,
  };
}
```

Data JavaScript is synchronous. Return plain JSON-compatible data. Promises are not supported.

## Duplicate Keys

These files conflict because both try to define `data.site`:

```txt
data/site.json
data/site.yaml
```

These also conflict because one file tries to define `data.shop` while another tries to define `data.shop.products`:

```txt
data/shop.json
data/shop/products.json
```

Veta fails the build instead of guessing which value should win.

## Data Versus File API

Use `data/` for global data that should be loaded once and shared everywhere.

Use the JavaScript file API for content collections and project files you want to enumerate manually:

```js
const posts = files.listFiles("content/posts/**/*.md");
```
