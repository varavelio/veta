---
title: "CLI Reference"
description: "Reference for every Veta command and command-line flag."
---

# CLI Reference

## `veta`

Shows help when called without arguments:

```sh
veta
```

## `veta init`

Creates a starter project.

```sh
veta init [PATH]
```

Examples:

```sh
veta init
veta init my-site
```

Flags:

```txt
--force    overwrite starter files that already exist
```

## `veta dev`

Starts the local development server with live reload.

```sh
veta dev [--config FILE]
```

Examples:

```sh
veta dev
veta dev --config ./veta.yaml
```

Host, port, and additional watched paths are configured in `veta.yaml` under `dev`.

## `veta build`

Builds the site for production.

```sh
veta build [--config FILE]
```

Examples:

```sh
veta build
veta build --config ./config/veta.yaml
```

## `veta version`

Prints version information.

```sh
veta version
veta --version
veta -v
```

## Config Discovery

`veta build` and `veta dev` search from the current directory upward for:

```txt
veta.yaml
veta.yml
.veta.yaml
.veta.yml
```

Use `--config` to bypass discovery.
