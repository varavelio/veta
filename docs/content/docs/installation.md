---
title: "Installation"
description: "Install the Veta CLI with shell installers, Homebrew, npm, Docker, or manual binaries."
---

# Installation

Veta is distributed as prebuilt binaries through GitHub Releases. The installers and package integrations download those release assets instead of rebuilding Veta locally.

## Linux And macOS

Use the shell installer:

```sh
curl -fsSL https://get.varavel.com/veta | sh
```

Or install with Homebrew:

```sh
brew install varavelio/tap/veta
```

## Windows

Use the PowerShell installer:

```powershell
irm https://get.varavel.com/veta.ps1 | iex
```

## npm

Install globally:

```sh
npm install --global @varavel/veta
```

Or install as a project development dependency:

```sh
npm install --save-dev @varavel/veta
```

The npm package installs the matching Veta binary for your platform during `postinstall`.

## Docker

Run Veta from Docker:

```sh
docker run --rm varavel/veta --help
```

Mount your project when you want to build it from a container:

```sh
docker run --rm -v "$PWD:/site" -w /site varavel/veta build
```

Images are also published to GitHub Container Registry as `ghcr.io/varavelio/veta`.

## Manual Download

Download archives from GitHub Releases:

```txt
https://github.com/varavelio/veta/releases
```

Release archives include Linux, macOS, and Windows binaries for supported architectures. Releases also publish `checksums.txt` and `manifest.json`.

## Verify Installation

Run:

```sh
veta --version
veta --help
```

You should see version information and the available commands.
