---
title: "Deployment"
description: "Deploy Veta's static output to any static hosting provider."
---

# Deployment

Veta produces static files. There is no production server requirement.

Build the site:

```sh
veta build
```

Deploy the configured output directory, usually `dist/`.

## Generic Static Hosting

Any host that can serve static files can serve a Veta site:

```txt
dist/
  index.html
  about/index.html
  styles.css
  robots.txt
```

Upload the contents of `dist/` to your host.

## CI Builds

A typical CI job only needs to install Veta and run:

```sh
veta build
```

Then publish `dist/` as the static artifact.

## npm Projects

If Veta is installed as a development dependency, add scripts:

```json
{
  "scripts": {
    "dev": "veta dev",
    "build": "veta build"
  },
  "devDependencies": {
    "@varavel/veta": "latest"
  }
}
```

## Docker Builds

You can build with Docker by mounting the project:

```sh
docker run --rm -v "$PWD:/site" -w /site varavel/veta build
```

## Production Reminder

Do not run `veta dev` in production. It is a local development workflow that serves a temporary output directory and injects live reload scripts into served HTML responses.
