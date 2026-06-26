// Package cli parses Veta command-line arguments.
//
// The package owns command selection, flag parsing, help output, and delegation
// to application workflows. It intentionally does not load configuration,
// discover pages, render content, process themes, or write output directly.
package cli
