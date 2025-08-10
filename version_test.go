package eol

import "testing"

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
		name     string
	}{
		{"1.24.6", "1.24", "semantic version with patch"},
		{"2.1.0", "2.1", "semantic version with zero patch"},
		{"10.15.7", "10.15", "double digit versions"},
		{"1.24", "1.24", "already major.minor format"},
		{"2.1", "2.1", "already major.minor format"},
		{"1.24.6-rc1", "1.24", "semantic version with prerelease"},
		{"1.24.6+build123", "1.24", "semantic version with build metadata"},
		{"1.24.6-rc1+build123", "1.24", "semantic version with prerelease and build"},
		{"v1.24.6", "v1.24.6", "version with v prefix (not normalized)"},
		{"1.24.6.7", "1.24.6.7", "four part version (not normalized)"},
		{"latest", "latest", "non-numeric version"},
		{"stable", "stable", "named version"},
		{"", "", "empty string"},
		{"  1.24.6  ", "1.24", "version with whitespace"},
		{"  1.24  ", "1.24", "major.minor with whitespace"},
		{"0.0.0", "0.0", "zero version"},
		{"1.0.0", "1.0", "major version one"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := normalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsSemanticVersion(t *testing.T) {
	t.Parallel()

	//nolint:govet // ok
	tests := []struct {
		input    string
		expected bool
		name     string
	}{
		{"1.24.6", true, "standard semantic version"},
		{"2.1.0", true, "semantic version with zero patch"},
		{"10.15.7", true, "double digit versions"},
		{"1.24.6-rc1", true, "semantic version with prerelease"},
		{"1.24.6+build123", true, "semantic version with build metadata"},
		{"1.24.6-rc1+build123", true, "semantic version with prerelease and build"},
		{"1.24", false, "major.minor only"},
		{"2.1", false, "major.minor only"},
		{"v1.24.6", false, "version with v prefix"},
		{"1.24.6.7", false, "four part version"},
		{"latest", false, "non-numeric version"},
		{"stable", false, "named version"},
		{"", false, "empty string"},
		{"1", false, "single number"},
		{"1.24.x", false, "version with x placeholder"},
		{"  1.24.6  ", true, "semantic version with whitespace"},
		{"0.0.0", true, "zero semantic version"},
		{"1.0.0-alpha", true, "semantic version with alpha prerelease"},
		{"1.0.0-alpha.1", true, "semantic version with numbered alpha"},
		{"1.0.0+20130313144700", true, "semantic version with timestamp build"},
		{"1.0.0-beta+exp.sha.5114f85", true, "semantic version with beta and sha"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isSemanticVersion(tt.input)
			if result != tt.expected {
				t.Errorf("isSemanticVersion(%q) = %t, expected %t", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractMajorMinor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
		name     string
	}{
		{"1.24.6", "1.24", "semantic version with patch"},
		{"2.1.0", "2.1", "semantic version with zero patch"},
		{"10.15.7", "10.15", "double digit versions"},
		{"1.24.6-rc1", "1.24", "semantic version with prerelease"},
		{"1.24.6+build123", "1.24", "semantic version with build metadata"},
		{"1.24.6-rc1+build123", "1.24", "semantic version with prerelease and build"},
		{"1.24", "1.24", "non-semantic version (major.minor)"},
		{"latest", "latest", "non-semantic version (named)"},
		{"v1.24.6", "v1.24.6", "version with prefix (not extracted)"},
		{"  1.24.6  ", "1.24", "semantic version with whitespace"},
		{"", "", "empty string"},
		{"0.0.0", "0.0", "zero semantic version"},
		{"999.999.999", "999.999", "large version numbers"},
		{"1.24.6.7", "1.24.6.7", "four part version (unchanged)"},
		{"1.0.0-alpha.1+build.2", "1.0", "complex semantic version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := extractMajorMinor(tt.input)
			if result != tt.expected {
				t.Errorf("extractMajorMinor(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestVersionPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		versions []string
		expected bool
	}{
		{
			name: "valid semantic versions",
			versions: []string{
				"1.0.0",
				"10.20.30",
				"1.0.0-alpha",
				"1.0.0-alpha.1",
				"1.0.0-0.3.7",
				"1.0.0-x.7.z.92",
				"1.0.0+20130313144700",
				"1.0.0-beta+exp.sha.5114f85",
				"0.0.1",
				"999.999.999",
			},
			pattern:  "semver",
			expected: true,
		},
		{
			name: "valid major.minor versions",
			versions: []string{
				"1.0",
				"10.20",
				"0.1",
				"999.999",
			},
			pattern:  "majorMinor",
			expected: true,
		},
		{
			name: "invalid semantic versions",
			versions: []string{
				"1",
				"1.0",
				"1.0.0.0",
				"v1.0.0",
				"1.0.0-",
				"1.0.0+",
				"latest",
				"stable",
				"",
				"1.0.0.",
				".1.0.0",
				"1..0.0",
			},
			pattern:  "semver",
			expected: false,
		},
		{
			name: "invalid major.minor versions",
			versions: []string{
				"1",
				"1.0.0",
				"v1.0",
				"latest",
				"",
				"1.",
				".1",
				"1..0",
			},
			pattern:  "majorMinor",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, version := range tt.versions {
				var matches bool

				switch tt.pattern {
				case "semver":
					matches = semverPattern.MatchString(version)
				case "majorMinor":
					matches = majorMinorPattern.MatchString(version)
				default:
					t.Fatalf("Unknown pattern: %s", tt.pattern)
				}

				if matches != tt.expected {
					if tt.expected {
						t.Errorf("Expected %q to match %s pattern", version, tt.pattern)
					} else {
						t.Errorf("Expected %q to NOT match %s pattern", version, tt.pattern)
					}
				}
			}
		})
	}
}

func TestVersionEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expected any
		name     string
		function string
		input    string
	}{
		{
			name:     "extractMajorMinor with complex prerelease",
			function: "extractMajorMinor",
			input:    "1.24.6-alpha.1.2.3+build.metadata.here",
			expected: "1.24",
		},
		{
			name:     "extractMajorMinor with only build metadata",
			function: "extractMajorMinor",
			input:    "1.24.6+very.long.build.metadata.12345",
			expected: "1.24",
		},
		{
			name:     "isSemanticVersion with minimal valid version",
			function: "isSemanticVersion",
			input:    "0.0.0",
			expected: true,
		},
		{
			name:     "isSemanticVersion with large numbers",
			function: "isSemanticVersion",
			input:    "999999.999999.999999",
			expected: true,
		},
		{
			name:     "normalizeVersion already normalized",
			function: "normalizeVersion",
			input:    "1.24",
			expected: "1.24",
		},
		{
			name:     "normalizeVersion zero version",
			function: "normalizeVersion",
			input:    "0.0.0",
			expected: "0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result any

			switch tt.function {
			case "extractMajorMinor":
				result = extractMajorMinor(tt.input)
			case "isSemanticVersion":
				result = isSemanticVersion(tt.input)
			case "normalizeVersion":
				result = normalizeVersion(tt.input)
			default:
				t.Fatalf("Unknown function: %s", tt.function)
			}

			if result != tt.expected {
				t.Errorf("%s(%q) = %v, expected %v", tt.function, tt.input, result, tt.expected)
			}
		})
	}
}
