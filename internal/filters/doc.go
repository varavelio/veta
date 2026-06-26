// Package filters builds Veta template filter functions.
//
// The package owns native filter definitions and loading user filter scripts
// from the filters directory through an injected script runner. It intentionally
// does not know about template engines, pages, components, themes, or output
// files.
package filters
