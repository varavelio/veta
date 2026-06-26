// Package render composes Veta page documents in memory.
//
// The package owns the page rendering sequence through injected dependencies. It
// intentionally does not load pages, load global data, scan components, register
// filters, resolve themes, copy assets, or write output files.
package render
