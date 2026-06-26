// Package pages discovers Veta page generator output.
//
// The package owns loading scripts from the project's pages directory,
// validating their returned page contract, normalizing permalinks, and detecting
// output path collisions. It intentionally does not know about templates,
// Markdown, components, themes, public assets, or output writing.
package pages
