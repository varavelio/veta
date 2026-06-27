package version

import "strings"

const (
	name = "veta"

	// Repository is the canonical Veta repository URL.
	Repository = "https://github.com/varavelio/veta"
)

var (
	// Version is the Veta version set at build time using ldflags.
	Version = "dev"

	// Commit is the git commit hash set at build time using ldflags.
	Commit = "unknown"

	// Date is the build date set at build time using ldflags.
	Date = "unknown"
)

// String returns the concise Veta version string.
func String() string {
	return name + " " + Number()
}

// Detailed returns Veta version metadata useful for diagnostics.
func Detailed() string {
	parts := []string{String()}
	if known(Commit) {
		parts = append(parts, "commit "+CommitHash())
	}
	if known(Date) {
		parts = append(parts, "built "+BuildDate())
	}

	return strings.Join(parts, " ")
}

// Number returns the normalized Veta version number.
func Number() string {
	return normalized(Version, "dev")
}

// CommitHash returns the normalized git commit hash.
func CommitHash() string {
	return normalized(Commit, "unknown")
}

// BuildDate returns the normalized build date.
func BuildDate() string {
	return normalized(Date, "unknown")
}

// normalized returns fallback when value is empty after trimming.
func normalized(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

// known reports whether a build metadata value should be displayed.
func known(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value != "unknown"
}
