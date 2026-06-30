// Package output writes Veta build files to disk.
//
// The package owns output path validation, output directory writing, optional
// generated HTML minification, cleaning, and copying public assets. It
// intentionally does not know about pages, templates, Markdown, filters,
// components, themes, or data loading.
package output
