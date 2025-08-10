package eol

import (
	"errors"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validate    func(*testing.T, *Config)
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "default config with valid command",
			args: []string{"products"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Format != FormatText {
					t.Errorf("Expected default format FormatText, got %v", c.Format)
				}
				if !c.CacheEnabled {
					t.Error("Expected cache to be enabled by default")
				}
				if c.CacheTTL != time.Hour {
					t.Errorf("Expected default cache TTL 1h, got %v", c.CacheTTL)
				}
				if c.Command != "products" {
					t.Errorf("Expected command 'products', got %s", c.Command)
				}
				if len(c.Args) != 0 {
					t.Errorf("Expected no args, got %v", c.Args)
				}
			},
		},
		{
			name: "command with arguments",
			args: []string{"product", "ubuntu"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Command != "product" {
					t.Errorf("Expected command 'product', got %s", c.Command)
				}
				if len(c.Args) != 1 || c.Args[0] != "ubuntu" {
					t.Errorf("Expected args ['ubuntu'], got %v", c.Args)
				}
			},
		},
		{
			name:        "empty args should error",
			args:        []string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config, err := NewConfig(tt.args...)
			if tt.expectError {
				if err == nil {
					t.Error("Expected NewConfig to return error")
				}

				return
			}

			if err != nil {
				t.Fatalf("NewConfig returned unexpected error: %v", err)
			}

			if config == nil {
				t.Fatal("NewConfig returned nil config")
			}

			tt.validate(t, config)
		})
	}
}

func TestConfigParseGlobalFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validate    func(*testing.T, *Config)
		name        string
		args        []string
		remaining   []string
		expectError bool
	}{
		{
			name:      "short flag json",
			args:      []string{"-f", "json", "products"},
			remaining: []string{"products"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Format != FormatJSON {
					t.Errorf("Expected format JSON, got %v", c.Format)
				}
			},
		},
		{
			name:      "long flag json",
			args:      []string{"--format", "json", "product", "ubuntu"},
			remaining: []string{"product", "ubuntu"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Format != FormatJSON {
					t.Errorf("Expected format JSON, got %v", c.Format)
				}
			},
		},
		{
			name:      "text format explicit",
			args:      []string{"-f", "text", "index"},
			remaining: []string{"index"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Format != FormatText {
					t.Errorf("Expected format text, got %v", c.Format)
				}
			},
		},
		{
			name:      "no format flag defaults to text",
			args:      []string{"products"},
			remaining: []string{"products"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Format != FormatText {
					t.Errorf("Expected default format text, got %v", c.Format)
				}
			},
		},
		{
			name:      "disable cache",
			args:      []string{"--disable-cache", "products"},
			remaining: []string{"products"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.CacheEnabled {
					t.Error("Expected cache to be disabled")
				}
			},
		},
		{
			name:      "cache directory",
			args:      []string{"--cache-dir", "/tmp/cache", "products"},
			remaining: []string{"products"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.CacheDir != "/tmp/cache" {
					t.Errorf("Expected cache dir /tmp/cache, got %s", c.CacheDir)
				}
			},
		},
		{
			name:      "cache TTL",
			args:      []string{"--cache-for", "2h", "products"},
			remaining: []string{"products"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.CacheTTL != 2*time.Hour {
					t.Errorf("Expected cache TTL 2h, got %v", c.CacheTTL)
				}
			},
		},
		{
			name:      "template directory",
			args:      []string{"--template-dir", "/tmp/templates", "product", "go"},
			remaining: []string{"product", "go"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.TemplateDir != "/tmp/templates" {
					t.Errorf("Expected template dir /tmp/templates, got %s", c.TemplateDir)
				}
			},
		},
		{
			name:      "short inline template",
			args:      []string{"-t", "{{ .Name }}", "product", "go"},
			remaining: []string{"product", "go"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.InlineTemplate != "{{ .Name }}" {
					t.Errorf("Expected inline template '{{ .Name }}', got %s", c.InlineTemplate)
				}
			},
		},
		{
			name:      "long inline template",
			args:      []string{"--template", "{{ .Version }}", "latest", "go"},
			remaining: []string{"latest", "go"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.InlineTemplate != "{{ .Version }}" {
					t.Errorf("Expected inline template '{{ .Version }}', got %s", c.InlineTemplate)
				}
			},
		},
		{
			name:      "combined flags",
			args:      []string{"-f", "json", "--disable-cache", "-t", "{{ .Name }}", "product", "go"},
			remaining: []string{"product", "go"},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Format != FormatJSON {
					t.Errorf("Expected format JSON, got %v", c.Format)
				}
				if c.CacheEnabled {
					t.Error("Expected cache to be disabled")
				}
				if c.InlineTemplate != "{{ .Name }}" {
					t.Errorf("Expected inline template '{{ .Name }}', got %s", c.InlineTemplate)
				}
			},
		},
		{
			name:        "missing format value",
			args:        []string{"-f"},
			expectError: true,
		},
		{
			name:        "invalid format",
			args:        []string{"-f", "xml", "products"},
			expectError: true,
		},
		{
			name:        "missing cache dir value",
			args:        []string{"--cache-dir"},
			expectError: true,
		},
		{
			name:        "missing cache TTL value",
			args:        []string{"--cache-for"},
			expectError: true,
		},
		{
			name:        "invalid cache TTL",
			args:        []string{"--cache-for", "invalid", "products"},
			expectError: true,
		},
		{
			name:        "missing template dir value",
			args:        []string{"--template-dir"},
			expectError: true,
		},
		{
			name:        "missing template value",
			args:        []string{"-t"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &Config{
				Format:       FormatText,
				CacheEnabled: true,
				CacheTTL:     time.Hour,
			}

			remaining, err := c.ParseGlobalFlags(tt.args)
			if tt.expectError {
				if err == nil {
					t.Error("Expected ParseGlobalFlags to return error")
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseGlobalFlags returned unexpected error: %v", err)
			}

			if len(remaining) != len(tt.remaining) {
				t.Errorf("Expected remaining args %v, got %v", tt.remaining, remaining)
			} else {
				for i, arg := range tt.remaining {
					if remaining[i] != arg {
						t.Errorf("Expected remaining arg %d to be %s, got %s", i, arg, remaining[i])
					}
				}
			}

			if tt.validate != nil {
				tt.validate(t, c)
			}
		})
	}
}

func TestConfigValidateArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		errorType   error
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "valid index command",
			args: []string{"index"},
		},
		{
			name: "valid products command",
			args: []string{"products"},
		},
		{
			name: "valid products with full flag",
			args: []string{"products", "--full"},
		},
		{
			name: "valid product command",
			args: []string{"product", "ubuntu"},
		},
		{
			name: "valid release command",
			args: []string{"release", "go", "1.24"},
		},
		{
			name: "valid latest command",
			args: []string{"latest", "ubuntu"},
		},
		{
			name: "valid categories command",
			args: []string{"categories"},
		},
		{
			name: "valid categories with arg",
			args: []string{"categories", "os"},
		},
		{
			name: "valid tags command",
			args: []string{"tags"},
		},
		{
			name: "valid tags with arg",
			args: []string{"tags", "canonical"},
		},
		{
			name: "valid identifiers command",
			args: []string{"identifiers"},
		},
		{
			name: "valid cache command",
			args: []string{"cache", "stats"},
		},
		{
			name: "valid templates command",
			args: []string{"templates"},
		},
		{
			name:        "empty args",
			args:        []string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateArgs(tt.args)
			if tt.expectError {
				if err == nil {
					t.Error("Expected validateArgs to return error")
					return
				}

				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error to be %v, got %v", tt.errorType, err)
				}

				return
			}

			if err != nil {
				t.Errorf("validateArgs returned unexpected error: %v", err)
			}
		})
	}
}

func TestConfigIsJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   OutputFormat
		expected bool
	}{
		{
			name:     "json format",
			format:   FormatJSON,
			expected: true,
		},
		{
			name:     "text format",
			format:   FormatText,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &Config{Format: tt.format}
			result := config.IsJSON()

			if result != tt.expected {
				t.Errorf("IsJSON() = %t, expected %t for format %d", result, tt.expected, tt.format)
			}
		})
	}
}

func TestConfigIsText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   OutputFormat
		expected bool
	}{
		{
			name:     "text format",
			format:   FormatText,
			expected: true,
		},
		{
			name:     "json format",
			format:   FormatJSON,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &Config{Format: tt.format}
			result := config.IsText()

			if result != tt.expected {
				t.Errorf("IsText() = %t, expected %t for format %d", result, tt.expected, tt.format)
			}
		})
	}
}

func TestConfigHasInlineTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		inlineTemplate string
		expected       bool
	}{
		{
			name:           "empty template",
			inlineTemplate: "",
			expected:       false,
		},
		{
			name:           "whitespace only template",
			inlineTemplate: "   ",
			expected:       true,
		},
		{
			name:           "simple template",
			inlineTemplate: "{{ .Name }}",
			expected:       true,
		},
		{
			name:           "complex template",
			inlineTemplate: "{{ .Name }}: {{ .Category }} ({{ join .Tags \", \" }})",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &Config{InlineTemplate: tt.inlineTemplate}
			result := config.HasInlineTemplate()

			if result != tt.expected {
				t.Errorf("HasInlineTemplate() = %t, expected %t for template %q", result, tt.expected, tt.inlineTemplate)
			}
		})
	}
}

func TestConfigHasCustomTemplateDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		templateDir string
		expected    bool
	}{
		{
			name:        "empty directory",
			templateDir: "",
			expected:    false,
		},
		{
			name:        "whitespace only directory",
			templateDir: "   ",
			expected:    true,
		},
		{
			name:        "relative path",
			templateDir: "./templates",
			expected:    true,
		},
		{
			name:        "absolute path",
			templateDir: "/home/user/.config/eol/templates",
			expected:    true,
		},
		{
			name:        "home directory shortcut",
			templateDir: "~/.config/eol/templates",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &Config{TemplateDir: tt.templateDir}
			result := config.TemplateDir != ""

			if result != tt.expected {
				t.Errorf("TemplateDir != \"\" = %t, expected %t for directory %q", result, tt.expected, tt.templateDir)
			}
		})
	}
}

func TestOutputFormatConstants(t *testing.T) {
	t.Parallel()

	if FormatText != 0 {
		t.Errorf("Expected FormatText to be 0, got %d", FormatText)
	}

	if FormatJSON != 1 {
		t.Errorf("Expected FormatJSON to be 1, got %d", FormatJSON)
	}
}

func TestConfigAllFields(t *testing.T) {
	t.Parallel()

	config := &Config{
		Command:        "product",
		Args:           []string{"ubuntu"},
		CacheDir:       "/tmp/cache",
		TemplateDir:    "/tmp/templates",
		InlineTemplate: "{{ .Name }}",
		CacheTTL:       2 * time.Hour,
		Format:         FormatJSON,
		CacheEnabled:   false,
	}

	if config.Command != "product" {
		t.Errorf("Expected Command to be 'product', got %q", config.Command)
	}

	if len(config.Args) != 1 || config.Args[0] != "ubuntu" {
		t.Errorf("Expected Args to be ['ubuntu'], got %v", config.Args)
	}

	if config.CacheDir != "/tmp/cache" {
		t.Errorf("Expected CacheDir to be '/tmp/cache', got %q", config.CacheDir)
	}

	if config.TemplateDir != "/tmp/templates" {
		t.Errorf("Expected TemplateDir to be '/tmp/templates', got %q", config.TemplateDir)
	}

	if config.InlineTemplate != "{{ .Name }}" {
		t.Errorf("Expected InlineTemplate to be '{{ .Name }}', got %q", config.InlineTemplate)
	}

	if config.CacheTTL != 2*time.Hour {
		t.Errorf("Expected CacheTTL to be 2h, got %v", config.CacheTTL)
	}

	if config.Format != FormatJSON {
		t.Errorf("Expected Format to be FormatJSON, got %d", config.Format)
	}

	if config.CacheEnabled {
		t.Errorf("Expected CacheEnabled to be false")
	}

	// Test helper methods.
	if !config.IsJSON() {
		t.Error("Expected IsJSON() to be true")
	}

	if config.IsText() {
		t.Error("Expected IsText() to be false")
	}

	if !config.HasInlineTemplate() {
		t.Error("Expected HasInlineTemplate() to be true")
	}

	if config.TemplateDir == "" {
		t.Error("Expected TemplateDir to be non-empty")
	}
}

func TestConfigZeroValues(t *testing.T) {
	t.Parallel()

	config := &Config{}

	if config.Format != FormatText {
		t.Errorf("Expected zero value format to be FormatText (%d), got %d", FormatText, config.Format)
	}

	if config.CacheEnabled {
		t.Error("Expected zero value CacheEnabled to be false")
	}

	if config.CacheTTL != 0 {
		t.Errorf("Expected zero value CacheTTL to be 0, got %v", config.CacheTTL)
	}

	if config.IsJSON() {
		t.Error("Expected IsJSON() to be false for zero value")
	}

	if !config.IsText() {
		t.Error("Expected IsText() to be true for zero value")
	}

	if config.HasInlineTemplate() {
		t.Error("Expected HasInlineTemplate() to be false for zero value")
	}

	if config.TemplateDir != "" {
		t.Error("Expected TemplateDir to be empty for zero value")
	}
}
