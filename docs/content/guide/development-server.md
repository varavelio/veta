---
title: "Development Server"
description: "Use veta dev for local development with temporary output and SSE live reload."
---

# Development Server

`veta dev` starts a local development server with live reload.

```sh
veta dev
```

By default, it serves at:

```txt
http://127.0.0.1:3000/
```

## Config

Configure the server in `veta.yaml`:

```yaml
dev:
  host: 127.0.0.1
  port: 3000
  watch:
    - content
```

`host` changes the address Veta binds to.

`port` changes the port.

`watch` adds project-relative files or directories to the watcher. Directories are watched recursively.

The only `veta dev` CLI flag is `--config`, which works like `veta build --config`:

```sh
veta dev --config path/to/veta.yaml
```

## Temporary Output

The dev server does not write to `build.output`. Instead, it creates an OS temporary directory, builds the site there, serves that directory, and removes it on shutdown.

This means running `veta dev` will not create or modify `dist/`.

## Rebuilds

On startup, Veta performs a full build. When relevant project files change, it performs another full clean rebuild into the temporary directory.

Veta watches:

```txt
veta.yaml
veta.yml
.veta.yaml
.veta.yml
pages/
data/
templates/
includes/
components/
filters/
public/
```

It also watches every path configured in `dev.watch`.

The watcher is intentionally simple and predictable. It rebuilds the whole site rather than trying to cache partial work.

Changes to `dev.host`, `dev.port`, or `dev.watch` require restarting `veta dev`, because those values define the running server and watcher.

## Live Reload

Veta uses Server-Sent Events at:

```txt
/_veta/live
```

When serving generated `.html` files, the dev server injects a small live reload script into the HTTP response. The script is not written to disk and never appears in production builds.

## Not A Production Server

`veta dev` is for local development only. For production, run:

```sh
veta build
```

Then deploy the output directory to a static hosting provider.
