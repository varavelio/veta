package js

import "strings"

const (
	defaultExportIdentifier          = "__vetaDefaultExport"
	defaultExportDefinedIdentifier   = "__vetaDefaultExportDefined"
	defaultExportDuplicateIdentifier = "__vetaDefaultExportDuplicate"

	defaultExportSyntax     = "export default"
	defaultExportAssignment = "__vetaExport.default ="
)

const exportDefaultPolyfill = `
	var __vetaDefaultExport;
	var __vetaDefaultExportDefined = false;
	var __vetaDefaultExportDuplicate = false;
	var __vetaExport = {};

	Object.defineProperty(__vetaExport, "default", {
		set: function(value) {
			if (__vetaDefaultExportDefined) {
				__vetaDefaultExportDuplicate = true;
				return;
			}

			__vetaDefaultExportDefined = true;
			__vetaDefaultExport = value;
		}
	});
`

// buildProgramSource instruments source code with Veta's export-default shim.
func buildProgramSource(source Source) string {
	// This is intentionally not a full ESM transform. Veta only supports the
	// exact `export default` syntax as sugar for its controlled default-export
	// capture mechanism.
	return exportDefaultPolyfill + "\n" + strings.ReplaceAll(
		source.Code,
		defaultExportSyntax,
		defaultExportAssignment,
	)
}
