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
- `internal/config` owns loading Veta tool configuration only. Site content, theme data, SEO metadata, navigation, and other user-facing values belong outside `veta.yaml`.
- `internal/data` owns loading global files from `data/` only. It keeps the directory flat, supports JSON/YAML/TOML/JS inputs, returns JSON-compatible values keyed by file stem, and must not render templates, discover pages, parse Markdown, or write output files.
- `internal/pages` owns loading generator scripts from `pages/` only. It keeps the directory flat, validates the page contract, normalizes permalinks, detects output path collisions, and must not render templates, parse Markdown, process components, resolve themes, copy assets, or write output files.
- `internal/js` owns synchronous execution of self-contained Veta JavaScript files and the `Veta` runtime object. It must not become a module loader, template engine, page loader, or output writer.
- `internal/markdown` owns Markdown-to-HTML conversion only. It must not know about pages, templates, filters, components, themes, data loading, or output files.
- `internal/theme` owns resolving configured theme sources into filesystems and composing them with project files. It must not load data, discover pages, render templates, execute JavaScript, process Markdown, or write output files.
- `internal/components` owns component discovery and source rewriting only. It discovers component templates from `components/`, parses registered tags in content, supports slots through injected renderers, and must not parse Markdown, own template engines, discover pages, load data, or write output files.
- `internal/filters` owns native filter definitions and loading JavaScript filter sources from `filters/` only. It keeps the directory flat, exposes filters through package-local interfaces, and must not import JavaScript runtimes, template engines, Markdown renderers, page loaders, or output writers directly.
- `internal/tmpl` owns Pongo2 template loading, template-name resolution, rendering, and filter registration. It must not know about pages, data loading, themes, Markdown, components, or output files.
- `internal/render` owns composing one page into one output document. It depends on injected interfaces for templates, content processors, and Markdown rendering, and must not discover files, load data, load filters, process themes, copy assets, or write output files.
- `internal/output` owns writing rendered files and copying `public/` assets to the output directory only. It validates relative output paths, detects collisions, and must not render templates, process Markdown, execute JavaScript, discover pages, load data, or resolve themes.
- `internal/vfs` owns virtual filesystem overlay and filtering helpers. It must not know about configuration, themes, templates, JavaScript, Markdown, Tailwind, or output writing.
- `internal/build` owns orchestrating one full build by wiring internal packages together through adapters. It must not absorb package-specific logic from config, theme, data, pages, templates, components, filters, Markdown, rendering, or output.
- `internal/cli` owns command selection, flag parsing, help text, and delegation to application workflows. It must not load site files, render content, resolve themes, or write build output directly.

## Testing & Quality

- Before considering any task complete, run `task ci`, which executes all project checks.
- Verify there are no errors. If there is any error, fix it and continue until the task is complete and `task ci` passes.

## Operational Commands

- Use `task --list-all` as the source of truth for available project commands.
- Do not duplicate or hard-code the command list here.
