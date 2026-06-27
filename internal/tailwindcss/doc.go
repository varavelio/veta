// Package tailwindcss builds CSS with the Tailwind CSS standalone CLI.
//
// The package owns materializing the embedded Tailwind executable into the Veta
// cache, preparing a temporary filesystem for class scanning, and returning the
// generated CSS as an output file. It intentionally does not load Veta
// configuration, resolve themes, render pages, or write final build output.
package tailwindcss
