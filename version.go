package eol

import (
	"regexp"
	"strings"
)

var (
	// SemverPattern matches semantic versions like "1.24.6", "2.1.0", "10.15.7", etc.
	semverPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-[0-9A-Za-z\-\.]+)?(?:\+[0-9A-Za-z\-\.]+)?$`)

	// MajorMinorPattern matches major.minor versions like "1.24", "2.1", etc.
	majorMinorPattern = regexp.MustCompile(`^(\d+)\.(\d+)$`)
)

// normalizeVersion attempts to normalize a version string for API compatibility.
// If the version looks like a semantic version (x.y.z), it returns the major.minor part (x.y).
// Otherwise, it returns the original version unchanged.
func normalizeVersion(version string) (ver string) {
	ver = strings.TrimSpace(version)
	if majorMinorPattern.MatchString(ver) {
		return
	}

	if matches := semverPattern.FindStringSubmatch(ver); matches != nil {
		return matches[1] + "." + matches[2]
	}

	return
}

// isSemanticVersion checks if a version string follows semantic versioning pattern.
func isSemanticVersion(version string) bool {
	return semverPattern.MatchString(strings.TrimSpace(version))
}

// extractMajorMinor extracts the major.minor part from a semantic version
// Returns the original string if it's not a semantic version.
func extractMajorMinor(version string) (ver string) {
	ver = strings.TrimSpace(version)
	if matches := semverPattern.FindStringSubmatch(ver); matches != nil {
		return matches[1] + "." + matches[2]
	}

	return
}
