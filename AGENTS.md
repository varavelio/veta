# Agent Context for veta

## Summary

This file defines the project-wide operating rules for AI agents working in this repository. Keep it concise, durable, and focused on guidance that helps future agents work correctly.

## Maintaining this Document

After completing any task, review this file and update it if you made structural changes or discovered patterns worth documenting. Only add information that helps understand how to work with the project. Avoid implementation details, file listings, or trivial changes. This is a general guide, not a changelog.

When updating this document, do so with the context of the entire document in mind; do not simply add new sections at the end, but place them where they make the most sense within the context of the document.

## General Instructions

You MUST follow the following instructions:

- At the start of every new assigned task or request, first run `task --list-all` to get the full current list of project commands. Do not hard-code the command list in this document because it will change over time.
- Whenever a new task or request is assigned, keep working without stopping until the task is fully completed.
- All written code must be professional, idiomatic, readable, and maintainable. Maintainability and readability are the top priority.
- Every function that is written must be properly documented with idiomatic English Godoc comments.

## Architecture Notes

- Keep `internal` packages narrow and composable. They should own one concern and avoid becoming build orchestrators.
- `internal/config` owns loading Veta tool configuration only, including build behavior such as output directory, clean mode, debug mode, format-specific settings such as HTML and Tailwind minification, and theme source. Site content, theme data, SEO metadata, navigation, and other user-facing values belong outside `veta.yaml`.
- `internal/data` owns loading global files from `data/` only. It supports nested directories, JSON/YAML/TOML/JS inputs, returns JSON-compatible values keyed by the directory/file stem path, and must not render templates, discover pages, parse Markdown, or write output files.
- `internal/pages` owns loading generator scripts from `pages/` only. It keeps the directory flat, validates the page contract, normalizes permalinks, detects output path collisions, and must not render templates, parse Markdown, process components, resolve themes, copy assets, or write output files. Page objects require `permalink`; `content` defaults to an empty string when omitted, and `template` is optional and is written relative to `templates/` without the `templates/` prefix.
- `internal/js` owns synchronous execution of self-contained Veta JavaScript files and the explicit runtime context object passed to default exports. It must not expose runtime APIs through global JavaScript variables or become a module loader, template engine, page loader, or output writer.
- `internal/dirs` owns resolving Veta runtime directories such as `~/.veta` and cache paths. It must not download themes, load project files, parse configuration, render content, or write build output.
- `internal/markdown` owns Markdown-to-HTML conversion only. It must not know about pages, templates, filters, components, themes, data loading, or output files.
- `internal/permalink` owns permalink normalization and project path to permalink conversion. It must not know about pages, output writing, templates, JavaScript runtimes, or project configuration.
- `internal/theme` owns resolving configured local and GitHub theme sources into filesystems, using `internal/dirs` cache paths for remote themes, and composing themes with project files. It must not load data, discover pages, render templates, execute JavaScript, process Markdown, or write output files.
- `internal/components` owns component discovery and source rewriting only. It discovers component templates from `components/`, parses registered tags in content, supports slots through injected renderers, and must not parse Markdown, own template engines, discover pages, load data, or write output files.
- `internal/filters` owns native filter definitions and loading JavaScript filter sources from `filters/` only. It keeps the directory flat, exposes filters through package-local interfaces, and must not import JavaScript runtimes, template engines, Markdown renderers, page loaders, or output writers directly.
- `internal/scaffold` owns writing embedded starter project files from `internal/scaffold/template` for `veta init` only. It must not parse CLI arguments, run builds, resolve themes, render pages, or load project data.
- `internal/template` owns Pongo2 template loading, template-name resolution, rendering, and filter registration. It must not know about pages, data loading, themes, Markdown, components, or output files.
- `internal/render` owns composing one page into one output document. It exposes the root template context `{ data, pages, page, props }`, where `pages` is the flat normalized page list and `page` is the current page. Pages without `template` emit raw `content`; pages with `template` process components/Markdown before rendering the template. It depends on injected interfaces for templates, content processors, and Markdown rendering, and must not discover files, load data, load filters, process themes, copy assets, or write output files.
- `internal/output` owns writing rendered files and copying `public/` assets to the output directory only. It validates relative output paths, detects collisions, applies final generated-file output transformations such as HTML-only minification, and must not render templates, process Markdown, execute JavaScript, discover pages, load data, or resolve themes.
- `internal/tailwindcss` owns running the embedded Tailwind CSS standalone CLI against a materialized output directory, reading the configured input stylesheet from an injected filesystem, and writing only the requested generated CSS file. It must not load Veta config, resolve themes, render pages, or write unrelated build output.
- `internal/version` owns Veta build metadata such as version, commit, and build date. It must not parse CLI arguments, render terminal UI, run builds, or inspect Git directly.
- `internal/vfs` owns virtual filesystem overlay and filtering helpers. It must not know about configuration, themes, templates, JavaScript, Markdown, Tailwind, or output writing.
- `internal/build` owns orchestrating one full build by discovering the Veta config from the working directory or an explicit config path, deriving the project root from that config file, and wiring internal packages together through adapters. It must not absorb package-specific logic from config, theme, data, pages, templates, components, filters, Markdown, rendering, or output.
- `internal/dev` owns the development-only server workflow: temporary output directories, full clean rebuild orchestration through `internal/build`, polling-based project watching, local HTTP serving, and SSE live reload. It must not become a production server, write to configured production output directories, duplicate build logic, or move CLI flag parsing out of `internal/cli`.
- `internal/cli` owns command selection, flag parsing, help text, human-facing command errors, and delegation to application workflows. Build behavior flags such as output directory, clean mode, and debug mode belong in `veta.yaml`, not the CLI. It must not load site files, render content, resolve themes, scaffold starter files, or write build output directly.

## Release & Distribution

- `scripts/release` owns local release artifact generation only: cross-compiling the CLI, injecting `internal/version` metadata, writing archives, and producing `manifest.json` plus `checksums.txt` in `dist/`. It must not publish releases, copy installers into `dist/`, or talk to package registries.
- GitHub Releases are the canonical source for published Veta binaries. Docker, Homebrew, and standalone installers should download release assets from GitHub Releases and verify release integrity rather than rebuilding Veta. The npm package should embed the release manifest generated from those assets and publish through npm trusted publishing/provenance, not long-lived npm tokens.
- `integrations/installers` owns distribution-channel wrappers and installer scripts. Keep channel-specific logic there, and do not move registry publishing, image publishing, or Homebrew tap updates into core `internal` packages.
- Release workflow jobs that need the project toolchain should run inside the existing devcontainer instead of recreating Go, Task, dprint, or lint tooling directly in GitHub Actions.

## Testing & Quality

- End-to-end tests live under `e2e/`, run through `task test:e2e`, and use real CLI executions against temporary projects. Keep reusable helpers in the e2e harness and put larger scenario inputs under `e2e/tests/<name>` instead of embedding sprawling project fixtures in test functions.
- For e2e fixtures that intentionally exercise Pongo control tags such as `{% block %}` and `{% extends %}`, prefer non-markup extensions like `.pongo` so dprint does not rewrite template directives as plain HTML.
- Before considering any task complete, run `task ci`, which executes all project checks.
- Verify there are no errors. If there is any error, fix it and continue until the task is complete and `task ci` passes.

## Documentation

- User-facing documentation lives under `docs/content/`. Every page should use frontmatter with `title` and `description`.

## Operational Commands

- Use `task --list-all` as the source of truth for available project commands.
- Do not duplicate or hard-code the command list here.
