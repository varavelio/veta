package sourcefile

import "regexp"

var allowedName = regexp.MustCompile(`^[A-Za-z0-9_]+\.[A-Za-z0-9_]+$`)

// Allowed reports whether name contains one extension separator and only ASCII
// alphanumeric or underscore characters around it.
func Allowed(name string) bool {
	return allowedName.MatchString(name)
}
