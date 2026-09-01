package page

import "regexp"

const (
	minIDLength = 16
	maxIDLength = 128
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ValidID reports whether id is a valid opaque page identifier.
func ValidID(id string) bool {
	return len(id) >= minIDLength && len(id) <= maxIDLength && idPattern.MatchString(id)
}
