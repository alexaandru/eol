package eol

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestTemplateManagerGetAvailableTemplates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		overrideDir       string
		setupUserTemplate bool
		expectError       bool
		minTemplates      int
	}{
		{
			name:         "embedded templates only",
			overrideDir:  "",
			minTemplates: 1, // Should have at least cache_stats template.
		},
		{
			name:              "with user templates",
			overrideDir:       "placeholder", // Will be replaced with temp dir.
			setupUserTemplate: true,
			minTemplates:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				tm      *TemplateManager
				tempDir string
				err     error
			)

			if tt.overrideDir != "" { //nolint:nestif // ok
				tempDir = t.TempDir()
				defer os.RemoveAll(tempDir)

				if tt.setupUserTemplate {
					// Create a test template file.
					templateContent := `{{.Name}} - {{.Label}}`

					err = os.WriteFile(filepath.Join(tempDir, "test.tmpl"), []byte(templateContent), 0o644)
					if err != nil {
						t.Fatalf("Failed to create test template: %v", err)
					}
				}

				tm, err = NewTemplateManager(tempDir, "", "", nil)
				if err != nil {
					t.Fatalf("Failed to create template manager: %v", err)
				}
			} else {
				tm, err = NewTemplateManager("", "", "", nil)
				if err != nil {
					t.Fatalf("Failed to create template manager: %v", err)
				}
			}

			templates := tm.GetAvailableTemplates()

			if len(templates) < tt.minTemplates {
				t.Errorf("Expected at least %d templates, got %d", tt.minTemplates, len(templates))
			}

			// Verify we get some expected templates.
			if !slices.Contains(templates, "cache_stats") {
				t.Error("Should contain 'cache_stats' template")
			}

			if tt.setupUserTemplate && !slices.Contains(templates, "test") {
				t.Error("Should contain user 'test' template")
			}
		})
	}
}

func TestGetEmbeddedTemplateContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		checkContent func([]byte) bool
		name         string
		path         string
		expectError  bool
	}{
		{
			name: "valid embedded template",
			path: "templates/cache_stats.tmpl",
			checkContent: func(content []byte) bool {
				return len(content) > 0 && strings.Contains(string(content), "{{")
			},
		},
		{
			name:        "nonexistent template",
			path:        "templates/nonexistent.tmpl",
			expectError: true,
		},
		{
			name:        "invalid path",
			path:        "invalid/path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, err := GetEmbeddedTemplateContent(tt.path)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.checkContent != nil && !tt.checkContent(content) {
				t.Error("Template content check failed")
			}
		})
	}
}

func TestTemplateManagerListTemplates(t *testing.T) {
	t.Parallel()

	tm, err := NewTemplateManager("", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	templates := tm.ListTemplates()
	if len(templates) == 0 {
		t.Error("Should return at least one template")
	}

	for _, tmpl := range templates {
		if tmpl.Name == "" {
			t.Error("Template name should not be empty")
		}

		if tmpl.Description == "" {
			t.Error("Template description should not be empty")
		}
	}

	found := false

	for _, tmpl := range templates {
		if tmpl.Name == "cache_stats" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Should include 'cache_stats' template")
	}
}

func TestTemplateManagerExportTemplates(t *testing.T) {
	t.Parallel()

	//nolint:govet // ok
	tests := []struct {
		name        string
		dir         string
		expectError bool
		setupFunc   func(*testing.T) string
	}{
		{
			name: "successful export",
		},
		{
			name:        "invalid directory",
			dir:         "/root/invalid/path/templates",
			expectError: true,
		},
		{
			name:        "permission denied directory",
			expectError: true,
			setupFunc: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()
				exportDir := filepath.Join(tempDir, "readonly")

				err := os.MkdirAll(exportDir, 0o755)
				if err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}

				err = os.Chmod(exportDir, 0o444)
				if err != nil {
					t.Fatalf("Failed to change directory permissions: %v", err)
				}

				t.Cleanup(func() { os.Chmod(exportDir, 0o755) })

				return filepath.Join(exportDir, "templates")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var exportDir string

			if tt.setupFunc != nil {
				exportDir = tt.setupFunc(t)
			} else {
				exportDir = tt.dir
				if exportDir == "" {
					exportDir = filepath.Join(t.TempDir(), "templates")
				}
			}

			tm, err := NewTemplateManager("", "", "", nil)
			if err != nil {
				t.Fatalf("Failed to create template manager: %v", err)
			}

			err = tm.ExportTemplates(exportDir)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			entries, err := os.ReadDir(exportDir)
			if err != nil {
				t.Errorf("Failed to read export directory: %v", err)
				return
			}

			if len(entries) == 0 {
				t.Error("No templates were exported")
			}

			cacheStatsPath := filepath.Join(exportDir, "cache_stats.tmpl")
			if _, statErr := os.Stat(cacheStatsPath); os.IsNotExist(statErr) {
				t.Error("cache_stats.tmpl should be exported")
			}
		})
	}
}

func TestExecuteInlineTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		template    string
		data        any
		expected    string
		expectError bool
	}{
		{
			name:     "simple field access",
			template: "{{.Name}}",
			data:     map[string]any{"Name": "Go"},
			expected: "Go",
		},
		{
			name:     "multiple fields",
			template: "{{.Name}} - {{.Version}}",
			data:     map[string]any{"Name": "Go", "Version": "1.21"},
			expected: "Go - 1.21",
		},
		{
			name:     "with join function",
			template: "{{join .Tags \", \"}}",
			data:     map[string]any{"Tags": []string{"lang", "google"}},
			expected: "lang, google",
		},
		{
			name:        "invalid template",
			template:    "{{.InvalidField}",
			data:        map[string]any{"Name": "Go"},
			expectError: true,
		},
		{
			name:     "missing field",
			template: "{{.MissingField}}",
			data:     map[string]any{"Name": "Go"},
			expected: "<no value>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ExecuteInlineTemplate(tt.template, tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestTemplateFunctions(t *testing.T) {
	t.Parallel()

	funcMap := getTemplateFuncMap()

	t.Run("join function", func(t *testing.T) {
		t.Parallel()

		join := funcMap["join"].(func([]string, string) string)
		result := join([]string{"a", "b", "c"}, ",")

		expected := "a,b,c"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}

		// Edge case: empty slice.
		result = join([]string{}, ",")
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}

		// Edge case: single element.
		result = join([]string{"single"}, ",")
		if result != "single" {
			t.Errorf("expected 'single', got %q", result)
		}
	})

	t.Run("toJSON function", func(t *testing.T) {
		t.Parallel()

		toJSON := funcMap["toJSON"].(func(any) string)

		// Test with simple object.
		data := map[string]string{"key": "value"}

		result := toJSON(data)
		if !strings.Contains(result, `"key": "value"`) {
			t.Errorf("expected JSON with key-value, got %q", result)
		}

		// Edge case: nil input.
		result = toJSON(nil)

		expected := null
		if !strings.Contains(result, expected) {
			t.Errorf("expected %q in result, got %q", expected, result)
		}

		// Edge case: invalid JSON data (channel - can't be marshaled).
		ch := make(chan int)

		result = toJSON(ch)
		if !strings.Contains(result, "error:") {
			t.Errorf("expected error message, got %q", result)
		}
	})

	t.Run("slice function", func(t *testing.T) {
		t.Parallel()

		slice := funcMap["slice"].(func(any, int, int) any)
		releases := []ProductRelease{
			{Name: "1.0.0"},
			{Name: "1.1.0"},
			{Name: "1.2.0"},
			{Name: "1.3.0"},
		}

		// Normal slice.
		result := slice(releases, 1, 3).([]ProductRelease)
		if len(result) != 2 {
			t.Errorf("expected 2 elements, got %d", len(result))
		}

		if result[0].Name != "1.1.0" || result[1].Name != "1.2.0" {
			t.Errorf("unexpected slice result: %v", result)
		}

		// Edge case: start out of bounds.
		result = slice(releases, 10, 15).([]ProductRelease)
		if len(result) != 0 {
			t.Errorf("expected empty slice for out of bounds start, got %d elements", len(result))
		}

		// Edge case: end beyond slice.
		result = slice(releases, 2, 10).([]ProductRelease)
		if len(result) != 2 {
			t.Errorf("expected 2 elements when end is beyond slice, got %d", len(result))
		}

		// Edge case: end <= start.
		result = slice(releases, 2, 1).([]ProductRelease)
		if len(result) != 0 {
			t.Errorf("expected empty slice when end <= start, got %d elements", len(result))
		}

		// Edge case: negative start.
		result = slice(releases, -1, 2).([]ProductRelease)
		if len(result) != 0 {
			t.Errorf("expected empty slice for negative start, got %d elements", len(result))
		}

		// Edge case: non-ProductRelease slice.
		stringSlice := []string{"a", "b", "c"}

		result2 := slice(stringSlice, 1, 2)
		if len(result2.([]string)) != len(stringSlice) {
			t.Errorf("expected original slice for non-ProductRelease type")
		}
	})

	t.Run("math functions", func(t *testing.T) {
		t.Parallel()

		sub := funcMap["sub"].(func(int, int) int)
		add := funcMap["add"].(func(int, int) int)
		div := funcMap["div"].(func(float64, float64) float64)
		mul := funcMap["mul"].(func(float64, float64) float64)

		if result := sub(10, 3); result != 7 {
			t.Errorf("sub(10, 3) expected 7, got %d", result)
		}

		if result := add(5, 3); result != 8 {
			t.Errorf("add(5, 3) expected 8, got %d", result)
		}

		if result := mul(3.5, 2.0); result != 7.0 {
			t.Errorf("mul(3.5, 2.0) expected 7.0, got %f", result)
		}

		if result := div(10.0, 2.0); result != 5.0 {
			t.Errorf("div(10.0, 2.0) expected 5.0, got %f", result)
		}

		if result := div(10.0, 0.0); result != 0.0 {
			t.Errorf("div(10.0, 0.0) expected 0.0, got %f", result)
		}

		if result := sub(5, 10); result != -5 {
			t.Errorf("sub(5, 10) expected -5, got %d", result)
		}
	})

	t.Run("default function", func(t *testing.T) {
		t.Parallel()

		defaultFunc := funcMap["default"].(func(any, any) any)

		// Test with nil value.
		result := defaultFunc("fallback", nil)
		if result != "fallback" {
			t.Errorf("expected 'fallback', got %v", result)
		}

		// Test with empty string.
		result = defaultFunc("fallback", "")
		if result != "fallback" {
			t.Errorf("expected 'fallback' for empty string, got %v", result)
		}

		// Test with nil string pointer.
		var nilStr *string

		result = defaultFunc("fallback", nilStr)
		if result != "fallback" {
			t.Errorf("expected 'fallback' for nil string pointer, got %v", result)
		}

		// Test with empty string pointer.
		emptyStr := ""

		result = defaultFunc("fallback", &emptyStr)
		if result != "fallback" {
			t.Errorf("expected 'fallback' for empty string pointer, got %v", result)
		}

		// Test with valid string.
		result = defaultFunc("fallback", "actual")
		if result != "actual" {
			t.Errorf("expected 'actual', got %v", result)
		}

		// Test with valid string pointer.
		validStr := "valid"

		result = defaultFunc("fallback", &validStr)
		if result != &validStr {
			t.Errorf("expected pointer to 'valid', got %v", result)
		}

		// Test with non-string type.
		result = defaultFunc("fallback", 42)
		if result != 42 {
			t.Errorf("expected 42, got %v", result)
		}
	})
}

func TestExecuteInlineTemplateExtended(t *testing.T) {
	t.Parallel()

	tm, err1 := NewTemplateManager("", "", "", nil)
	if err1 != nil {
		t.Fatalf("Failed to create template manager: %v", err1)
	}

	tests := []struct {
		data         any
		name         string
		templateStr  string
		expectOutput string
		expectError  bool
	}{
		{
			name:         "simple template",
			templateStr:  "Hello {{ .Name }}!",
			data:         map[string]string{"Name": "World"},
			expectError:  false,
			expectOutput: "Hello World!",
		},
		{
			name:         "template with function",
			templateStr:  "{{ add .A .B }}",
			data:         map[string]int{"A": 5, "B": 3},
			expectError:  false,
			expectOutput: "8",
		},
		{
			name:        "invalid template syntax",
			templateStr: "{{ .Name",
			data:        map[string]string{"Name": "Test"},
			expectError: true,
		},
		{
			name:         "template execution error",
			templateStr:  "{{ .NonExistent }}",
			data:         map[string]string{"Name": "Test"},
			expectError:  false,
			expectOutput: "<no value>",
		},
		{
			name:         "empty template",
			templateStr:  "",
			data:         map[string]string{"Name": "Test"},
			expectError:  false,
			expectOutput: "",
		},
		{
			name:        "template with complex data",
			templateStr: "{{ range .Items }}{{ .Name }}: {{ .Value }}\n{{ end }}",
			data: map[string][]map[string]string{
				"Items": {
					{"Name": "item1", "Value": "value1"},
					{"Name": "item2", "Value": "value2"},
				},
			},
			expectError:  false,
			expectOutput: "item1: value1\nitem2: value2\n",
		},
		{
			name:        "template with slice function",
			templateStr: "{{ $slice := slice .Releases 0 2 }}{{ len $slice }}",
			data: map[string][]ProductRelease{
				"Releases": {
					{Name: "1.0.0"},
					{Name: "1.1.0"},
					{Name: "1.2.0"},
				},
			},
			expectError:  false,
			expectOutput: "2",
		},
		{
			name:         "template with toJSON function",
			templateStr:  "{{ toJSON .Data }}",
			data:         map[string]map[string]string{"Data": {"key": "value"}},
			expectError:  false,
			expectOutput: "{\n    \"key\": \"value\"\n  }",
		},
		{
			name:         "template with default function",
			templateStr:  "{{ default \"N/A\" .MissingField }}",
			data:         map[string]string{"Name": "Test"},
			expectError:  false,
			expectOutput: "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := tm.ExecuteInline(tt.templateStr, tt.data)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				resultStr := string(result)
				if resultStr != tt.expectOutput {
					t.Errorf("expected %q, got %q", tt.expectOutput, resultStr)
				}
			}
		})
	}
}

func TestExecuteInlineTemplateEdgeCases(t *testing.T) {
	t.Parallel()

	tm, err1 := NewTemplateManager("", "", "", nil)
	if err1 != nil {
		t.Fatalf("Failed to create template manager: %v", err1)
	}

	t.Run("nil data", func(t *testing.T) {
		t.Parallel()

		result, err := tm.ExecuteInline("{{ . }}", nil)
		if err != nil {
			t.Errorf("unexpected error with nil data: %v", err)
		}

		if string(result) != "<no value>" {
			t.Errorf("expected '<no value>', got %q", string(result))
		}
	})

	t.Run("very long template", func(t *testing.T) {
		t.Parallel()

		longTemplate := strings.Repeat("{{ .Name }}", 1000)
		data := map[string]string{"Name": "X"}

		result, err := tm.ExecuteInline(longTemplate, data)
		if err != nil {
			t.Errorf("unexpected error with long template: %v", err)
		}

		expected := strings.Repeat("X", 1000)
		if string(result) != expected {
			t.Errorf("long template result mismatch")
		}
	})

	//nolint:gosmopolitan // intended
	t.Run("template with unicode", func(t *testing.T) {
		t.Parallel()

		tpl := "{{ .Message }}"
		data := map[string]string{"Message": "Hello 世界! 🌍"}

		result, err := tm.ExecuteInline(tpl, data)
		if err != nil {
			t.Errorf("unexpected error with unicode: %v", err)
		}

		if string(result) != "Hello 世界! 🌍" {
			t.Errorf("unicode not preserved: %q", string(result))
		}
	})

	t.Run("recursive template structure", func(t *testing.T) {
		t.Parallel()

		tpl := "{{ range .Items }}{{ range .SubItems }}{{ .Name }}{{ end }}{{ end }}"
		data := map[string][]map[string][]map[string]string{
			"Items": {
				{
					"SubItems": {
						{"Name": "A"},
						{"Name": "B"},
					},
				},
			},
		}

		result, err := tm.ExecuteInline(tpl, data)
		if err != nil {
			t.Errorf("unexpected error with recursive structure: %v", err)
		}

		if string(result) != "AB" {
			t.Errorf("expected 'AB', got %q", string(result))
		}
	})
}

func TestTemplateFunctionEdgeCases(t *testing.T) {
	t.Parallel()

	funcMap := getTemplateFuncMap()

	t.Run("exit function", func(t *testing.T) {
		t.Parallel()

		// We can't actually test os.Exit as it would terminate the test
		// But we can verify the function exists and has correct type.
		exitFunc, exists := funcMap["exit"]
		if !exists {
			t.Error("exit function should exist in funcMap")
		}

		// Verify it's callable (though we won't actually call it).
		if exitFunc == nil {
			t.Error("exit function should not be nil")
		}
	})

	t.Run("slice function with edge cases", func(t *testing.T) {
		t.Parallel()

		slice := funcMap["slice"].(func(any, int, int) any)
		emptyReleases := []ProductRelease{}

		result := slice(emptyReleases, 0, 1).([]ProductRelease)
		if len(result) != 0 {
			t.Errorf("expected empty result for empty input, got %d elements", len(result))
		}

		// Test with equal start and end.
		releases := []ProductRelease{{Name: "1.0.0"}}

		result = slice(releases, 0, 0).([]ProductRelease)
		if len(result) != 0 {
			t.Errorf("expected empty result for equal start/end, got %d elements", len(result))
		}

		// Test with struct type (non-slice).
		nonSlice := "not a slice"

		result2 := slice(nonSlice, 0, 1)
		if result2 != nonSlice {
			t.Error("expected original value for non-slice input")
		}
	})

	t.Run("default function with complex types", func(t *testing.T) {
		t.Parallel()

		defaultFunc := funcMap["default"].(func(any, any) any)

		result := defaultFunc("fallback", []string{"item"})
		if len(result.([]string)) != 1 {
			t.Error("expected slice to be returned as-is")
		}

		testMap := map[string]int{"key": 42}
		result = defaultFunc("fallback", testMap)

		resultMap, ok := result.(map[string]int)
		if !ok || len(resultMap) != 1 || resultMap["key"] != 42 {
			t.Error("expected map to be returned as-is")
		}

		result = defaultFunc("fallback", 0)
		if result != 0 {
			t.Error("expected zero int to be returned as-is")
		}

		result = defaultFunc("fallback", false)
		if result != false {
			t.Error("expected false bool to be returned as-is")
		}
	})

	t.Run("math functions with extreme values", func(t *testing.T) {
		t.Parallel()

		div := funcMap["div"].(func(float64, float64) float64)
		mul := funcMap["mul"].(func(float64, float64) float64)

		// Test with very large numbers.
		result := mul(1e10, 1e10)
		if result != 1e20 {
			t.Errorf("expected 1e20, got %e", result)
		}

		// Test with very small numbers.
		result = div(1e-10, 1e-5)

		expected := 1e-5
		if result-expected > 1e-15 || expected-result > 1e-15 {
			t.Errorf("expected %e, got %e", expected, result)
		}

		// Test with infinity.
		result = div(1.0, 0.0)
		if result != 0.0 {
			t.Errorf("expected 0.0 for division by zero, got %f", result)
		}
	})
}

func TestGetTemplateFuncMap(t *testing.T) {
	t.Parallel()

	funcMap := getTemplateFuncMap()

	// Test that required functions are present.
	expectedFuncs := []string{"join"}

	for _, funcName := range expectedFuncs {
		if _, exists := funcMap[funcName]; !exists {
			t.Errorf("Function %q should be present in funcMap", funcName)
		}
	}

	// Test join function.
	if joinFunc, ok := funcMap["join"]; ok {
		if fn, fnOk := joinFunc.(func([]string, string) string); fnOk {
			result := fn([]string{"a", "b", "c"}, ", ")

			expected := "a, b, c"
			if result != expected {
				t.Errorf("join function: expected %q, got %q", expected, result)
			}
		} else {
			t.Error("join function has wrong signature")
		}
	}
}

func TestTemplateManagerLoadFromFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		templateName  string
		content       string
		setupFile     bool
		expectError   bool
		useInvalidDir bool
	}{
		{
			name:         "no override directory",
			templateName: "test",
			expectError:  true,
		},
		{
			name:         "successful load",
			templateName: "valid",
			content:      "Hello {{.Name}}!",
			setupFile:    true,
			expectError:  false,
		},
		{
			name:         "file not found",
			templateName: "missing",
			expectError:  true,
		},
		{
			name:         "invalid template syntax",
			templateName: "invalid",
			content:      "{{.Name",
			setupFile:    true,
			expectError:  true,
		},
		{
			name:          "non-existent directory",
			templateName:  "test",
			useInvalidDir: true,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				tm  *TemplateManager
				err error
			)

			switch {
			case tt.name == "no override directory":
				tm, err = NewTemplateManager("", "", "", nil)
				if err != nil {
					t.Fatalf("Failed to create template manager: %v", err)
				}
			case tt.useInvalidDir:
				_, err = NewTemplateManager("/non/existent/directory", "", "", nil)
				if err == nil {
					t.Error("Expected error for non-existent directory")
				}

				return
			default:
				tempDir := t.TempDir()
				defer os.RemoveAll(tempDir)

				tm, err = NewTemplateManager(tempDir, "", "", nil)
				if err != nil {
					t.Fatalf("Failed to create template manager: %v", err)
				}

				if tt.setupFile {
					templatePath := filepath.Join(tempDir, tt.templateName+".tmpl")

					err = os.WriteFile(templatePath, []byte(tt.content), 0o644)
					if err != nil {
						t.Fatalf("Failed to create test template: %v", err)
					}
				}
			}

			tmpl, err := tm.loadFromFile(tt.templateName)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tmpl == nil {
				t.Error("Template should not be nil")
			}
		})
	}
}

func TestTemplateManagerExecute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data         any
		checkOutput  func(string) bool
		name         string
		templateName string
		expectError  bool
	}{
		{
			name:         "execute cache_stats template",
			templateName: "cache_stats",
			data:         map[string]any{"Name": "Go", "Label": "Go Programming Language"},
			checkOutput: func(output string) bool {
				return strings.Contains(output, "Cache Statistics")
			},
		},
		{
			name:         "nonexistent template",
			templateName: "nonexistent",
			data:         map[string]any{},
			expectError:  true,
		},
		{
			name:         "execute products template",
			templateName: "products",
			data:         map[string]any{"Result": []any{}},
			checkOutput: func(output string) bool {
				return strings.Contains(output, "Products")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tm, err := NewTemplateManager("", "", "", nil)
			if err != nil {
				t.Fatalf("Failed to create template manager: %v", err)
			}

			output, err := tm.Execute(tt.templateName, tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.checkOutput != nil && !tt.checkOutput(string(output)) {
				t.Errorf("Output check failed: %q", string(output))
			}
		})
	}
}

func TestTemplateManagerAddUserTemplates(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	// Create test template files.
	templateFiles := map[string]string{
		"user1.tmpl": "{{.Name}}",
		"user2.tmpl": "{{.Label}}",
		"notmpl.txt": "not a template", // Should be ignored.
	}

	for filename, content := range templateFiles {
		writeErr := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0o644)
		if writeErr != nil {
			t.Fatalf("Failed to write template file %s: %v", filename, writeErr)
		}
	}

	tm, err := NewTemplateManager(tempDir, "", "", nil)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	// This method is private, but we can test it indirectly through GetAvailableTemplates.
	availableTemplates := tm.GetAvailableTemplates()

	// Check that user templates were found.
	if !slices.Contains(availableTemplates, "user1") {
		t.Error("Should contain user1 template")
	}

	if !slices.Contains(availableTemplates, "user2") {
		t.Error("Should contain user2 template")
	}
	// Notmpl.txt should not be included as it doesn't have .tmpl extension.
	if slices.Contains(availableTemplates, "notmpl") {
		t.Error("Should not contain notmpl (non-.tmpl file)")
	}
}

func TestTemplateManagerExecuteTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		templateName string
		expectError  bool
	}{
		{
			name:         "existing template",
			templateName: "cache_stats",
			expectError:  false,
		},
		{
			name:         "nonexistent template",
			templateName: "nonexistent",
			expectError:  true,
		},
		{
			name:         "template execution error - missing template",
			templateName: "completely_nonexistent_template",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tm, err := NewTemplateManager("", "", "", nil)
			if err != nil {
				t.Fatalf("Failed to create template manager: %v", err)
			}

			_, err = tm.Execute(tt.templateName, map[string]any{})
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
		})
	}
}

func TestTemplateManagerInlineTemplate(t *testing.T) {
	t.Parallel()

	//nolint:govet // ok
	tests := []struct {
		name           string
		command        string
		args           []string
		inlineTemplate string
		data           any
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "inline template for products command",
			command:        "products",
			args:           []string{},
			inlineTemplate: "Product count: {{len .Result}}",
			data:           map[string]any{"Result": []string{"go", "python", "rust"}},
			expectedOutput: "Product count: 3",
		},
		{
			name:           "inline template for cache stats",
			command:        "cache",
			args:           []string{"stats"},
			inlineTemplate: "Cache has {{.TotalFiles}} files",
			data:           map[string]any{"TotalFiles": 42},
			expectedOutput: "Cache has 42 files",
		},
		{
			name:           "inline template overrides builtin",
			command:        "product",
			args:           []string{"go"},
			inlineTemplate: "{{.Name}} ({{.Category}})",
			data:           map[string]any{"Name": "Go", "Category": "lang"},
			expectedOutput: "Go (lang)",
		},
		{
			name:           "invalid inline template returns error",
			command:        "products",
			args:           []string{},
			inlineTemplate: "{{.InvalidField",
			data:           map[string]any{"Result": []string{}, "Total": 0},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tm, err := NewTemplateManager("", tt.inlineTemplate, tt.command, tt.args)
			if err != nil && !tt.expectError {
				t.Fatalf("Failed to create template manager: %v", err)
			}

			if err == nil && tt.expectError {
				t.Error("Expected error but got none")
				return
			}

			if tt.expectError {
				return
			}

			templateName := getTemplateNameForCommand(tt.command, tt.args)
			if templateName == "" {
				t.Skip("No template for this command combination")
			}

			output, err := tm.Execute(templateName, tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if string(output) != tt.expectedOutput {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, string(output))
			}

			expectedSource := "inline"
			if strings.Contains(tt.name, "invalid") {
				expectedSource = "builtin"
			}

			if tm.GetTemplateSource(templateName) != expectedSource {
				t.Errorf("Expected template source to be %q, got %q", expectedSource, tm.GetTemplateSource(templateName))
			}
		})
	}
}

func TestGetTemplateNameForCommand(t *testing.T) {
	t.Parallel()

	//nolint:govet // ok
	tests := []struct {
		name     string
		command  string
		args     []string
		expected string
	}{
		{
			name:     "index command",
			command:  "index",
			args:     []string{},
			expected: "index",
		},
		{
			name:     "products command",
			command:  "products",
			args:     []string{},
			expected: "products",
		},
		{
			name:     "products full command",
			command:  "products",
			args:     []string{"full"},
			expected: "product_details",
		},
		{
			name:     "product command",
			command:  "product",
			args:     []string{"go"},
			expected: "product_details",
		},
		{
			name:     "release command",
			command:  "release",
			args:     []string{"go", "1.20"},
			expected: "product_release",
		},
		{
			name:     "latest command",
			command:  "latest",
			args:     []string{"go"},
			expected: "product_release",
		},
		{
			name:     "categories command",
			command:  "categories",
			args:     []string{},
			expected: "categories",
		},
		{
			name:     "categories with category",
			command:  "categories",
			args:     []string{"lang"},
			expected: "products_by_category",
		},
		{
			name:     "tags command",
			command:  "tags",
			args:     []string{},
			expected: "tags",
		},
		{
			name:     "tags with tag",
			command:  "tags",
			args:     []string{"google"},
			expected: "products_by_tag",
		},
		{
			name:     "identifiers command",
			command:  "identifiers",
			args:     []string{},
			expected: "identifiers",
		},
		{
			name:     "identifiers with type",
			command:  "identifiers",
			args:     []string{"cpe"},
			expected: "identifiers_by_type",
		},
		{
			name:     "cache stats command",
			command:  "cache",
			args:     []string{"stats"},
			expected: "cache_stats",
		},
		{
			name:     "cache clear command",
			command:  "cache",
			args:     []string{"clear"},
			expected: "",
		},
		{
			name:     "cache without subcommand",
			command:  "cache",
			args:     []string{},
			expected: "",
		},
		{
			name:     "templates command",
			command:  "templates",
			args:     []string{},
			expected: "templates",
		},
		{
			name:     "templates export command",
			command:  "templates",
			args:     []string{"export", "/tmp"},
			expected: "template_export",
		},
		{
			name:     "unknown command",
			command:  "unknown",
			args:     []string{},
			expected: "",
		},
		{
			name:     "empty command",
			command:  "",
			args:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := getTemplateNameForCommand(tt.command, tt.args)
			if result != tt.expected {
				t.Errorf("getTemplateNameForCommand(%q, %v) = %q, expected %q",
					tt.command, tt.args, result, tt.expected)
			}
		})
	}
}

func TestTemplateManagerOverrideTemplateErrors(t *testing.T) {
	t.Parallel()

	t.Run("invalid override template syntax", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create an invalid template file.
		invalidTemplate := filepath.Join(tempDir, "invalid.tmpl")

		err := os.WriteFile(invalidTemplate, []byte("{{.Name"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create invalid template file: %v", err)
		}

		// This should fail during construction due to invalid template.
		_, err = NewTemplateManager(tempDir, "", "", nil)
		if err == nil {
			t.Error("Expected error for invalid template syntax")
		}

		if !strings.Contains(err.Error(), "unclosed action") {
			t.Errorf("Expected 'unclosed action' error, got: %v", err)
		}
	})

	t.Run("permission denied on override directory", func(t *testing.T) {
		t.Parallel()

		// Try to use a directory that doesn't exist and can't be created.
		_, err := NewTemplateManager("/root/nonexistent", "", "", nil)
		if err == nil {
			t.Error("Expected error for permission denied directory")
		}

		if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("Expected 'permission denied' error, got: %v", err)
		}
	})
}

func TestTemplateFuncMapEdgeCases(t *testing.T) {
	t.Parallel()

	funcMap := getTemplateFuncMap()

	t.Run("slice function with invalid input", func(t *testing.T) {
		t.Parallel()

		sliceFunc := funcMap["slice"].(func(any, int, int) any)

		// Test with non-slice input.
		result := sliceFunc("not a slice", 0, 1)
		if result != "not a slice" {
			t.Errorf("Expected input to be returned unchanged for non-slice, got %v", result)
		}
	})

	t.Run("slice function with out of bounds", func(t *testing.T) {
		t.Parallel()

		sliceFunc := funcMap["slice"].(func(any, int, int) any)
		releases := []ProductRelease{{Name: "1.0"}, {Name: "2.0"}}

		result := sliceFunc(releases, 5, 10) // Start index out of bounds.
		if len(result.([]ProductRelease)) != 0 {
			t.Errorf("Expected empty slice for out of bounds start, got %v", result)
		}
	})
}
