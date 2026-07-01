---
title: "Getting Started"
description: "Build your first Veta site from an empty folder to a production-ready static output."
---

# Getting Started

This guide builds a small Veta site from scratch. By the end, you will understand the project structure, the page generator model, templates, data files, components, Markdown content, Tailwind CSS, the development server, and the production build.

## 1. Install Veta

Install Veta with any of the methods documented in [Installation](./installation.md).

Verify the CLI is available:

```sh
veta --version
```

## 2. Create A Project

Create a starter project:

```sh
veta init my-site
cd my-site
```

The starter contains these files:

```txt
my-site/
  veta.yaml
  components/
    note.html
  data/
    site.json
  pages/
    site.js
  public/
    robots.txt
    styles.css
  templates/
    base.html
```

Start the development server:

```sh
veta dev
```

Open the printed local URL. Veta builds the site into a temporary directory, serves it locally, watches your project files, and reloads the browser when a rebuild finishes. The development server does not write to `dist/`.

## 3. Understand The Config

The starter `veta.yaml` looks like this:

```yaml
build:
  output: dist
  clean: true

html:
  minify: true

dev:
  host: 127.0.0.1
  port: 3000
  watch: []

tailwindcss:
  stylesheets:
    - styles.css
  minify: true
```

The important defaults are:

- `build.output` is the directory written by `veta build`.
- `build.clean` removes the output directory before writing a new build.
- `html.minify` minifies generated `.html` files.
- `dev.host` configures the local development server host.
- `dev.port` configures the local development server port.
- `dev.watch` is an array of additional directories for the development server to watch, beyond Veta's own files and directories.
- `tailwindcss.stylesheets` points to Tailwind CSS entrypoints under `public/`.
- `tailwindcss.minify` minifies the generated stylesheet.

## 4. Edit Site Data

Open `data/site.json` and change the name or description:

```json
{
  "name": "My Veta Site",
  "description": "A small site built with Veta."
}
```

Data files become available in templates and page generators through the `data` object. The file `data/site.json` becomes `data.site`.

## 5. Generate Pages With JavaScript

Open `pages/site.js`:

```js
export default function({ data, files, httpClient }) {
  return [
    {
      permalink: "/",
      template: "base",
      title: "Home",
      content: "Welcome to **Veta**.",
    },
    {
      permalink: "/about/",
      template: "base",
      title: "About",
      content: "This page was generated from `pages/site.js`.",
    },
  ];
}
```

Every file in `pages/` must be a JavaScript file. It must export a default function that returns an array of page objects.

Each page object needs a `permalink`:

```js
{
  permalink: "/about/",
  template: "base",
  title: "About",
  content: "About this site."
}
```

If `template` is present, Veta processes the page `content` through components and Markdown, then renders the template from `templates/`. If `template` is omitted, Veta writes `content` as raw output.

## 6. Use A Template

Open `templates/base.html`:

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="description" content="{{ data.site.description }}">
    <title>{{ page.title }} - {{ data.site.name }}</title>
    <link rel="stylesheet" href="/styles.css">
  </head>
  <body>
    <nav>
      {% for item in pages %}
      <a href="{{ item.permalink }}">{{ item.title }}</a>
      {% endfor %}
    </nav>

    <main>
      <h1>{{ page.title }}</h1>
      {{ page.content }}
    </main>
  </body>
</html>
```

Templates receive four root values:

- `data`: global data loaded from `data/`.
- `pages`: every normalized page returned by your page generators.
- `page`: the current page.
- `props`: component props when rendering a component.

## 7. Add A Component

Components are templates stored in `components/`. The starter includes `components/note.html`:

```html
<aside class="rounded border border-zinc-200 bg-zinc-50 p-4">
  {{ props.content }}
</aside>
```

Use it inside page content:

```js
{
  permalink: "/",
  template: "base",
  title: "Home",
  content: "<note>This content supports **Markdown**.</note>",
}
```

The inner content of a paired component tag is rendered as Markdown before it is passed to the component as `props.content`.

## 8. Read Markdown Files

Create content files:

```txt
content/posts/hello.md
content/posts/second.md
```

Example Markdown file with YAML frontmatter:

```md
---
title: Hello World
date: "2026-06-30"
tags:
  - intro
---

# Hello World

This post is stored as Markdown.
```

Generate pages from those files:

```js
export default function({ files }) {
  const posts = files.listFiles("content/posts/**/*.md");

  return posts.map((path) => {
    const post = files.readMarkdownFile(path);

    return {
      permalink: files.toPermalink(path, { stripPrefix: "content" }),
      template: "base",
      title: post.frontmatter.title,
      content: post.content,
    };
  });
}
```

`readMarkdownFile` returns:

```js
{
  content: "# Hello World\n\nThis post is stored as Markdown.\n",
  frontmatter: { title: "Hello World", date: "2026-06-30", tags: ["intro"] },
  path: "content/posts/hello.md"
}
```

## 9. Add Styles With Tailwind CSS

The starter uses `public/styles.css` as the Tailwind entrypoint:

```css
@import "tailwindcss";
```

When `tailwindcss.stylesheets` includes `styles.css`, Veta reads `public/styles.css`, runs the embedded Tailwind CSS standalone CLI against the generated output, and writes `dist/styles.css`.

Use classes directly in templates and components:

```html
<body class="mx-auto max-w-2xl px-6 py-10 text-zinc-950">
```

## 10. Build For Production

Stop the dev server and run:

```sh
veta build
```

Veta writes the production site to `dist/` by default. Generated `.html` files are minified when `html.minify: true` is set. Public assets are copied from `public/` to the output root.

You can deploy `dist/` to any static host.

## Next Steps

Read these next:

- [Project Structure](./guide/project-structure.md)
- [Pages](./guide/pages.md)
- [Templates](./guide/templates.md)
- [JavaScript API](./api/javascript.md)
- [Build And Output](./guide/build-and-output.md)
