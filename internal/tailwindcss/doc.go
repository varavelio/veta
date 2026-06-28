// Package tailwindcss builds CSS with the Tailwind CSS standalone CLI.
//
// The package owns materializing the embedded Tailwind executable into the Veta
// cache, reading the configured input stylesheet from a provided filesystem, and
// running Tailwind against a materialized working directory. It intentionally
// does not load Veta configuration, resolve themes, render pages, or write final
// build output beyond the requested generated CSS file.
package tailwindcss
