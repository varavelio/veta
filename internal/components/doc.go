// Package components renders Veta component tags embedded in content strings.
//
// The package owns component discovery, tag parsing, attribute parsing, nested
// component expansion, and component template invocation through an injected
// renderer. It intentionally does not know about Markdown, pages, global data
// loading, themes, or output files.
package components
