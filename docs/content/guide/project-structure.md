---
title: "Project Structure"
description: "Learn what each Veta project directory does and which files are required."
---

# Project Structure

A Veta project is a folder with a `veta.yaml` configuration file and optional feature directories. The starter project created by `veta init` shows the common layout:

```txt
.
  veta.yaml
  components/
  data/
  filters/
  includes/
  pages/
  public/
  templates/
```

Only `veta.yaml` and `pages/` are necessary for most useful sites. The other directories are optional and can be introduced as the project grows.

## `veta.yaml`

`veta.yaml` configures Veta itself. It controls build output, clean mode, generated HTML minification, Tailwind CSS, and themes.

Site content does not belong in `veta.yaml`. Put content, navigation, SEO metadata, and theme data in `data/` or content files read through the JavaScript file API.

## `pages/`

`pages/` contains flat JavaScript page generator files. Each file must end in `.js` and export a default function that returns an array of page objects.

The directory is intentionally flat. Do not put nested folders under `pages/`.

## `templates/`

`templates/` contains Pongo templates used by page objects. A page object references templates relative to this directory:

```js
{
  permalink: "/",
  template: "base",
}
```

That can resolve `templates/base.html`, `templates/base.j2`, or another non-ignored file with the same stem.

## `includes/`

`includes/` is the standard place for shared Pongo fragments used by both templates and components:

```html
{% include "includes/brand.html" %}
```

Use `includes/` when markup is shared across templates and components. Markup used only by page templates belongs in `templates/`; markup used only as a content component belongs in `components/`.

Pongo can include project files from other directories too, but `includes/` is Veta's convention and is watched by `veta dev` by default.

## `components/`

`components/` contains reusable component templates. Component tags are derived from file paths:

```txt
components/note.j2       -> <note>
components/ui/card.j2    -> <ui-card>
```

Components are used inside page content and receive attributes through `props`.

## `data/`

`data/` contains global data files. Veta supports JSON, YAML, TOML, and JavaScript:

```txt
data/site.json              -> data.site
data/navigation.yaml        -> data.navigation
data/theme/colors.toml      -> data.theme.colors
```

Nested directories become nested keys.

## `filters/`

`filters/` contains custom JavaScript template filters. The directory is flat and every filter file must end in `.js`.

```txt
filters/titlecase.js        -> {{ page.title|titlecase }}
```

## `public/`

`public/` contains static files copied to the output root. For example:

```txt
public/robots.txt           -> dist/robots.txt
public/images/logo.svg      -> dist/images/logo.svg
public/styles.css           -> Tailwind entrypoint when configured
```

Public assets are copied as-is. Generated HTML minification applies only to generated page output, not to copied public files.
