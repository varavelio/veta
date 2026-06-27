// Package build orchestrates one Veta site build.
//
// The package connects the focused internal packages into an end-to-end build
// pipeline. It discovers the project config, derives the project root from that
// config file, and owns package adaptation and workflow ordering. Rendering,
// data loading, theme resolution, template execution, component processing, and
// output writing remain delegated to their specialized packages.
package build
