// Package build orchestrates one Veta site build.
//
// The package connects the focused internal packages into an end-to-end build
// pipeline. It owns package adaptation and workflow ordering, while rendering,
// data loading, theme resolution, template execution, component processing, and
// output writing remain delegated to their specialized packages.
package build
