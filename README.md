# Veta

<p>
  <a href="https://github.com/varavelio/veta/actions">
    <img src="https://github.com/varavelio/veta/actions/workflows/ci.yaml/badge.svg" alt="CI status"/>
  </a>
  <a href="https://github.com/varavelio/veta/releases/latest">
    <img src="https://img.shields.io/github/release/varavelio/veta.svg" alt="Release Version"/>
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/github/license/varavelio/veta.svg" alt="License"/>
  </a>
  <a href="https://github.com/varavelio/veta">
    <img src="https://img.shields.io/github/stars/varavelio/veta?style=flat&label=github+stars"/>
  </a>
</p>

<p>
  <a href="https://varavel.com">
    <img src="https://cdn.jsdelivr.net/gh/varavelio/brand@1.0.0/dist/badges/project.svg" alt="A Varavel project"/>
  </a>
</p>

Veta is a static site generator for small, scriptable sites. It combines flat
JavaScript page generators, Pongo2 templates, project data files, components,
filters, Markdown rendering, themes, and embedded Tailwind CSS into a single CLI.

## Install

Download binaries from GitHub Releases, or use one of the package installers once
they are published:

```sh
npm install --global @varavel/veta
brew install varavelio/tap/veta
docker run --rm ghcr.io/varavelio/veta --help
```

From source:

```sh
go install github.com/varavelio/veta/cmd/veta@latest
```

## Usage

Create a starter project:

```sh
veta init my-site
```

Build a site from the current project:

```sh
veta build
```

Build with an explicit config file:

```sh
veta build --config path/to/veta.yaml
```

## Project Structure

Veta projects can contain:

- `veta.yaml` for tool configuration.
- `data/` for JSON, YAML, TOML, or JavaScript data files.
- `pages/` for flat JavaScript page generator files.
- `templates/` for Pongo2 layouts.
- `components/` for reusable component templates.
- `filters/` for JavaScript filters.
- `public/` for static assets copied to the output directory.
