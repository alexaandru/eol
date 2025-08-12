package eol

import (
	"bytes"
	"context"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClientNormReleaseArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		errorType   string
		args        []string
		expected    []string
		expectError bool
	}{
		{
			name:        "valid args",
			args:        []string{"go", "1.24"},
			expected:    []string{"go", "1.24"},
			expectError: false,
		},
		{
			name:        "semantic version normalization",
			args:        []string{"go", "1.24.0"},
			expected:    []string{"go", "1.24"},
			expectError: false,
		},
		{
			name:        "missing product name",
			args:        []string{},
			expectError: true,
			errorType:   "product name and release name required",
		},
		{
			name:        "missing release name",
			args:        []string{"go"},
			expectError: true,
			errorType:   "product name and release name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "release", []string{})

			result, err := client.normReleaseArgs(tt.args)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorType != "" && !strings.Contains(err.Error(), tt.errorType) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorType, err)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d args, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected arg[%d] = %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestClientHandleRouting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		command     string
		errorType   string
		args        []string
		expectError bool
	}{
		{
			name:        "help command",
			command:     "help",
			args:        []string{},
			expectError: true,
			errorType:   "help requested",
		},
		{
			name:        "unknown command",
			command:     "unknown",
			args:        []string{},
			expectError: true,
			errorType:   "unknown command",
		},
		{
			name:        "cache missing subcommand",
			command:     "cache",
			args:        []string{},
			expectError: true,
			errorType:   "cache subcommand is required",
		},
		{
			name:        "templates export missing dir",
			command:     "templates",
			args:        []string{"export"},
			expectError: true,
			errorType:   "output directory is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, tt.command, tt.args)

			err := client.Handle()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorType != "" && !strings.Contains(err.Error(), tt.errorType) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorType, err)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestClientHandleIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		expectResponse bool
		expectHeader   bool
	}{
		{
			name:           "successful index",
			expectResponse: true,
			expectHeader:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "index", []string{})

			err := client.HandleIndex()
			if err != nil {
				t.Fatalf("HandleIndex() error = %v", err)
			}

			if tt.expectResponse {
				if client.response == nil {
					t.Error("Expected response to be set")
					return
				}

				if _, ok := client.response.(*IndexResponse); !ok {
					t.Errorf("Expected IndexResponse, got %T", client.response)
				}
			}

			if tt.expectHeader && client.responseHeader == "" {
				t.Error("Expected response header to be set")
			}
		})
	}
}

func TestClientHandleProducts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseType   string
		args           []string
		expectResponse bool
		expectHeader   bool
	}{
		{
			name:           "products list",
			args:           []string{},
			expectResponse: true,
			expectHeader:   false,
			responseType:   "ProductListResponse",
		},
		{
			name:           "products full",
			args:           []string{"--full"},
			expectResponse: true,
			expectHeader:   true,
			responseType:   "FullProductListResponse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "products", tt.args)

			err := client.HandleProducts()
			if err != nil {
				t.Fatalf("HandleProducts() error = %v", err)
			}

			if tt.expectResponse && client.response == nil {
				t.Error("Expected response to be set")
				return
			}

			if tt.expectHeader && client.responseHeader == "" {
				t.Error("Expected response header to be set")
			}

			switch tt.responseType {
			case "ProductListResponse":
				if _, ok := client.response.(*ProductListResponse); !ok {
					t.Errorf("Expected ProductListResponse, got %T", client.response)
				}
			case "FullProductListResponse":
				if _, ok := client.response.(*FullProductListResponse); !ok {
					t.Errorf("Expected FullProductListResponse, got %T", client.response)
				}
			}
		})
	}
}

func TestClientHandleProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		errorType      string
		args           []string
		expectError    bool
		expectResponse bool
		expectHeader   bool
	}{
		{
			name:           "valid product",
			args:           []string{"ubuntu"},
			expectError:    false,
			expectResponse: true,
			expectHeader:   true,
		},
		{
			name:        "missing product name",
			args:        []string{},
			expectError: true,
			errorType:   "product name is required",
		},
		{
			name:        "empty product name",
			args:        []string{""},
			expectError: true,
			errorType:   "product name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "product", tt.args)

			err := client.HandleProduct()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorType != "" && !strings.Contains(err.Error(), tt.errorType) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorType, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("HandleProduct() error = %v", err)
			}

			if tt.expectResponse {
				if client.response == nil {
					t.Error("Expected response to be set")
					return
				}

				if _, ok := client.response.(*ProductResponse); !ok {
					t.Errorf("Expected ProductResponse, got %T", client.response)
				}
			}

			if tt.expectHeader && client.responseHeader == "" {
				t.Error("Expected response header to be set")
			}
		})
	}
}

func TestClientHandleRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		errorType      string
		args           []string
		expectError    bool
		expectResponse bool
		expectHeader   bool
	}{
		{
			name:           "valid release",
			args:           []string{"go", "1.24"},
			expectError:    false,
			expectResponse: true,
			expectHeader:   true,
		},
		{
			name:        "missing args",
			args:        []string{},
			expectError: true,
			errorType:   "product name and release name required",
		},
		{
			name:        "insufficient args",
			args:        []string{"go"},
			expectError: true,
			errorType:   "product name and release name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "release", tt.args)

			err := client.HandleRelease()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorType != "" && !strings.Contains(err.Error(), tt.errorType) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorType, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("HandleRelease() error = %v", err)
			}

			if tt.expectResponse {
				if client.response == nil {
					t.Error("Expected response to be set")
					return
				}

				if _, ok := client.response.(*ProductReleaseResponse); !ok {
					t.Errorf("Expected ProductReleaseResponse, got %T", client.response)
				}
			}

			if tt.expectHeader && client.responseHeader == "" {
				t.Error("Expected response header to be set")
			}
		})
	}
}

func TestClientHandleLatest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		errorType      string
		args           []string
		expectError    bool
		expectResponse bool
		expectHeader   bool
	}{
		{
			name:           "valid latest",
			args:           []string{"ubuntu"},
			expectError:    false,
			expectResponse: true,
			expectHeader:   true,
		},
		{
			name:        "missing product name",
			args:        []string{},
			expectError: true,
			errorType:   "product name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "latest", tt.args)

			err := client.HandleLatest()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorType != "" && !strings.Contains(err.Error(), tt.errorType) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorType, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("HandleLatest() error = %v", err)
			}

			if tt.expectResponse {
				if client.response == nil {
					t.Error("Expected response to be set")
					return
				}

				if _, ok := client.response.(*ProductReleaseResponse); !ok {
					t.Errorf("Expected ProductReleaseResponse, got %T", client.response)
				}
			}

			if tt.expectHeader && client.responseHeader == "" {
				t.Error("Expected response header to be set")
			}
		})
	}
}

func TestClientHandleCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseType   string
		args           []string
		expectResponse bool
		expectHeader   bool
	}{
		{
			name:           "categories list",
			args:           []string{},
			expectResponse: true,
			expectHeader:   false,
			responseType:   "CategoriesResponse",
		},
		{
			name:           "category products",
			args:           []string{"lang"},
			expectResponse: true,
			expectHeader:   true,
			responseType:   "CategoryProductsResponse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "categories", tt.args)

			err := client.HandleCategories()
			if err != nil {
				t.Fatalf("HandleCategories() error = %v", err)
			}

			if tt.expectResponse && client.response == nil {
				t.Error("Expected response to be set")
				return
			}

			if tt.expectHeader && client.responseHeader == "" {
				t.Error("Expected response header to be set")
			}

			switch tt.responseType {
			case "CategoriesResponse":
				if _, ok := client.response.(*CategoriesResponse); !ok {
					t.Errorf("Expected CategoriesResponse, got %T", client.response)
				}
			case "CategoryProductsResponse":
				if _, ok := client.response.(*CategoryProductsResponse); !ok {
					t.Errorf("Expected CategoryProductsResponse, got %T", client.response)
				}
			}
		})
	}
}

func TestClientHandleTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseType   string
		args           []string
		expectResponse bool
		expectHeader   bool
	}{
		{
			name:           "tags list",
			args:           []string{},
			expectResponse: true,
			expectHeader:   false,
			responseType:   "TagsResponse",
		},
		{
			name:           "tag products",
			args:           []string{"google"},
			expectResponse: true,
			expectHeader:   true,
			responseType:   "TagProductsResponse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "tags", tt.args)

			err := client.HandleTags()
			if err != nil {
				t.Fatalf("HandleTags() error = %v", err)
			}

			if tt.expectResponse && client.response == nil {
				t.Error("Expected response to be set")
				return
			}

			if tt.expectHeader && client.responseHeader == "" {
				t.Error("Expected response header to be set")
			}

			switch tt.responseType {
			case "TagsResponse":
				if _, ok := client.response.(*TagsResponse); !ok {
					t.Errorf("Expected TagsResponse, got %T", client.response)
				}
			case "TagProductsResponse":
				if _, ok := client.response.(*TagProductsResponse); !ok {
					t.Errorf("Expected TagProductsResponse, got %T", client.response)
				}
			}
		})
	}
}

func TestClientHandleIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseType   string
		args           []string
		expectResponse bool
		expectHeader   bool
	}{
		{
			name:           "identifiers list",
			args:           []string{},
			expectResponse: true,
			expectHeader:   false,
			responseType:   "IdentifierTypesResponse",
		},
		{
			name:           "identifier type",
			args:           []string{"cpe"},
			expectResponse: true,
			expectHeader:   true,
			responseType:   "TypeIdentifiersResponse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "identifiers", tt.args)

			err := client.HandleIdentifiers()
			if err != nil {
				t.Fatalf("HandleIdentifiers() error = %v", err)
			}

			if tt.expectResponse && client.response == nil {
				t.Error("Expected response to be set")
				return
			}

			if tt.expectHeader && client.responseHeader == "" {
				t.Error("Expected response header to be set")
			}

			switch tt.responseType {
			case "IdentifierTypesResponse":
				if _, ok := client.response.(*IdentifierTypesResponse); !ok {
					t.Errorf("Expected IdentifierTypesResponse, got %T", client.response)
				}
			case "TypeIdentifiersResponse":
				if _, ok := client.response.(*TypeIdentifiersResponse); !ok {
					t.Errorf("Expected TypeIdentifiersResponse, got %T", client.response)
				}
			}
		})
	}
}

func TestClientHandleCacheStats(t *testing.T) {
	t.Parallel()

	responses := createMockResponses(t)
	client := createTestClient(t, t.Context(), responses, "cache", []string{"stats"})

	err := client.HandleCacheStats()
	if err != nil {
		t.Fatalf("HandleCacheStats() error = %v", err)
	}

	if client.response == nil {
		t.Error("Expected response to be set")
		return
	}

	if _, ok := client.response.(*CacheStats); !ok {
		t.Errorf("Expected CacheStats, got %T", client.response)
	}
}

func TestClientHandleCacheClear(t *testing.T) {
	t.Parallel()

	responses := createMockResponses(t)
	client := createTestClient(t, t.Context(), responses, "cache", []string{"clear"})

	var buf bytes.Buffer

	client.sink = &buf

	err := client.HandleCacheClear()
	if err != nil {
		t.Fatalf("HandleCacheClear() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Cache cleared successfully") {
		t.Errorf("Expected clear message, got: %s", output)
	}
}

func TestClientHandleTemplates(t *testing.T) {
	t.Parallel()

	responses := createMockResponses(t)
	client := createTestClient(t, t.Context(), responses, "templates", []string{})

	err := client.HandleTemplates()
	if err != nil {
		t.Fatalf("HandleTemplates() error = %v", err)
	}

	if client.response == nil {
		t.Error("Expected response to be set")
		return
	}

	if _, ok := client.response.(*TemplateListResponse); !ok {
		t.Errorf("Expected TemplateListResponse, got %T", client.response)
	}

	if client.responseHeader == "" {
		t.Error("Expected response header to be set")
	}
}

func TestClientHandleTemplateExport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		errorType   string
		args        []string
		expectError bool
	}{
		{
			name:        "valid export",
			args:        []string{"templates", "export", "/tmp/test"},
			expectError: false,
		},
		{
			name:        "missing output directory",
			args:        []string{"templates", "export"},
			expectError: true,
			errorType:   "output directory is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "templates", tt.args[1:])

			if len(tt.args) > 2 {
				// Create a temporary directory for the test.
				outputDir := t.TempDir()
				client.config.Args = []string{"export", outputDir}
			}

			err := client.HandleTemplateExport()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorType != "" && !strings.Contains(err.Error(), tt.errorType) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorType, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("HandleTemplateExport() error = %v", err)
			}

			if client.response == nil {
				t.Error("Expected response to be set")
				return
			}

			if _, ok := client.response.(*TemplateExportResponse); !ok {
				t.Errorf("Expected TemplateExportResponse, got %T", client.response)
			}

			if client.responseHeader == "" {
				t.Error("Expected response header to be set")
			}
		})
	}
}

func TestClientHandleCompletionAuto(t *testing.T) {
	t.Parallel()

	responses := createMockResponses(t)
	client := createTestClient(t, t.Context(), responses, "completion", []string{})

	err := client.HandleCompletionAuto()
	if err != nil {
		t.Fatalf("HandleCompletionAuto() error = %v", err)
	}

	if client.response == nil {
		t.Error("Expected response to be set")
		return
	}

	resp, ok := client.response.(*CompletionResponse)
	if !ok {
		t.Errorf("Expected CompletionResponse, got %T", client.response)
		return
	}

	if resp.Shell == "" {
		t.Error("Expected shell to be set")
	}

	if resp.Script == "" {
		t.Error("Expected script to be set")
	}

	if client.responseHeader == "" {
		t.Error("Expected response header to be set")
	}
}

func TestClientHandleCompletionBash(t *testing.T) {
	t.Parallel()

	responses := createMockResponses(t)
	client := createTestClient(t, t.Context(), responses, "completion", []string{"bash"})

	err := client.HandleCompletionBash()
	if err != nil {
		t.Fatalf("HandleCompletionBash() error = %v", err)
	}

	if client.response == nil {
		t.Error("Expected response to be set")
		return
	}

	resp, ok := client.response.(*CompletionResponse)
	if !ok {
		t.Errorf("Expected CompletionResponse, got %T", client.response)
		return
	}

	if resp.Shell != "bash" {
		t.Errorf("Expected shell to be 'bash', got %s", resp.Shell)
	}

	if resp.Script == "" {
		t.Error("Expected script to be set")
	}

	if client.responseHeader == "" {
		t.Error("Expected response header to be set")
	}
}

func TestClientHandleCompletionZsh(t *testing.T) {
	t.Parallel()

	responses := createMockResponses(t)
	client := createTestClient(t, t.Context(), responses, "completion", []string{"zsh"})

	err := client.HandleCompletionZsh()
	if err != nil {
		t.Fatalf("HandleCompletionZsh() error = %v", err)
	}

	if client.response == nil {
		t.Error("Expected response to be set")
		return
	}

	resp, ok := client.response.(*CompletionResponse)
	if !ok {
		t.Errorf("Expected CompletionResponse, got %T", client.response)
		return
	}

	if resp.Shell != "zsh" {
		t.Errorf("Expected shell to be 'zsh', got %s", resp.Shell)
	}

	if resp.Script == "" {
		t.Error("Expected script to be set")
	}

	if client.responseHeader == "" {
		t.Error("Expected response header to be set")
	}
}

func TestClientExecuteInlineTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		template     string
		data         any
		expectOutput string
		expectError  bool
	}{
		{
			name:         "simple template",
			template:     "{{.Name}}",
			data:         map[string]any{"Name": "test"},
			expectOutput: "test",
			expectError:  false,
		},
		{
			name:        "invalid template",
			template:    "{{.Invalid",
			data:        map[string]any{"Name": "test"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "index", []string{})

			var buf bytes.Buffer

			client.sink = &buf
			client.config.InlineTemplate = tt.template

			err := client.executeInlineTemplate(tt.data)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("executeInlineTemplate() error = %v", err)
			}

			output := buf.String()
			if output != tt.expectOutput {
				t.Errorf("Expected output %q, got %q", tt.expectOutput, output)
			}
		})
	}
}

func TestClientFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		response    any
		checkOutput func([]byte) bool
		name        string
		expectError bool
	}{
		{
			name:     "IndexResponse",
			response: &IndexResponse{UriListResponse: &UriListResponse{}},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name:     "CategoriesResponse",
			response: &CategoriesResponse{UriListResponse: &UriListResponse{}},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name:     "TagsResponse",
			response: &TagsResponse{UriListResponse: &UriListResponse{}},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name:     "IdentifierTypesResponse",
			response: &IdentifierTypesResponse{UriListResponse: &UriListResponse{}},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name:     "ProductListResponse",
			response: &ProductListResponse{},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name: "FullProductListResponse",
			response: &FullProductListResponse{
				Result: []ProductDetails{
					{
						Name:     "go",
						Label:    "Go",
						Category: "lang",
						Releases: []ProductRelease{},
					},
				},
			},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name: "ProductResponse",
			response: &ProductResponse{
				Result: ProductDetails{
					Name:     "go",
					Label:    "Go",
					Category: "lang",
				},
			},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name: "ProductReleaseResponse",
			response: &ProductReleaseResponse{
				Result: ProductRelease{
					Name:         "1.24",
					Label:        "1.24",
					ReleaseDate:  "2025-02-11",
					IsLts:        false,
					IsMaintained: true,
					IsEol:        false,
				},
			},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name: "CategoryProductsResponse",
			response: &CategoryProductsResponse{
				ProductListResponse: &ProductListResponse{},
				Category:            "lang",
			},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name: "TagProductsResponse",
			response: &TagProductsResponse{
				ProductListResponse: &ProductListResponse{},
				Tag:                 "google",
			},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name: "TypeIdentifiersResponse",
			response: &TypeIdentifiersResponse{
				IdentifierListResponse: &IdentifierListResponse{},
				Type:                   "cpe",
			},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name: "CacheStats",
			response: &CacheStats{
				TotalFiles: 5,
				TotalSize:  1048576, // 1MB in bytes.
			},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name: "TemplateListResponse",
			response: &TemplateListResponse{
				Templates: []TemplateInfo{},
				Total:     0,
			},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name: "TemplateExportResponse",
			response: &TemplateExportResponse{
				OutputDir: "/tmp/test",
				Message:   "Templates exported",
			},
			checkOutput: func(output []byte) bool {
				return len(output) > 0
			},
		},
		{
			name:     "CompletionResponse",
			response: &CompletionResponse{Shell: "bash", Script: "#!/bin/bash\necho test"},
			checkOutput: func(output []byte) bool {
				return string(output) == "#!/bin/bash\necho test"
			},
		},
		{
			name:        "unknown response type",
			response:    "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "index", []string{})

			output, err := client.Format(tt.response)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			if tt.checkOutput != nil && !tt.checkOutput(output) {
				t.Errorf("Output check failed for %s", tt.name)
			}
		})
	}
}

func TestClientExtractTemplateData(t *testing.T) {
	t.Parallel()

	responses := createMockResponses(t)
	client := createTestClient(t, t.Context(), responses, "index", []string{})

	//nolint:govet // ok
	tests := []struct {
		name         string
		response     any
		expectedType string
		checkFunc    func(t *testing.T, data any)
	}{
		{
			name:         "IndexResponse extracts UriListResponse",
			response:     &IndexResponse{UriListResponse: &UriListResponse{Result: []Uri{{Name: "test", URI: "/test"}}}},
			expectedType: "*eol.UriListResponse",
			checkFunc: func(t *testing.T, data any) {
				t.Helper()

				if resp, ok := data.(*UriListResponse); !ok {
					t.Errorf("Expected *UriListResponse, got %T", data)
				} else if len(resp.Result) != 1 || resp.Result[0].Name != "test" {
					t.Errorf("Expected result with 'test', got %v", resp.Result)
				}
			},
		},
		{
			name:         "ProductResponse extracts Result field",
			response:     &ProductResponse{Result: ProductDetails{Name: "go", Category: "lang"}},
			expectedType: "*eol.ProductDetails",
			checkFunc: func(t *testing.T, data any) {
				t.Helper()

				if resp, ok := data.(*ProductDetails); !ok {
					t.Errorf("Expected *ProductDetails, got %T", data)
				} else if resp.Name != "go" || resp.Category != "lang" {
					t.Errorf("Expected go/lang, got %s/%s", resp.Name, resp.Category)
				}
			},
		},
		{
			name:         "ProductReleaseResponse extracts Result field",
			response:     &ProductReleaseResponse{Result: ProductRelease{Name: "1.21", IsLts: true}},
			expectedType: "*eol.ProductRelease",
			checkFunc: func(t *testing.T, data any) {
				t.Helper()

				if resp, ok := data.(*ProductRelease); !ok {
					t.Errorf("Expected *ProductRelease, got %T", data)
				} else if resp.Name != "1.21" || !resp.IsLts {
					t.Errorf("Expected 1.21/true, got %s/%t", resp.Name, resp.IsLts)
				}
			},
		},
		{
			name:     "CategoryProductsResponse creates composite struct",
			response: &CategoryProductsResponse{ProductListResponse: &ProductListResponse{Result: []ProductSummary{{Name: "test"}}}, Category: "lang"},
			checkFunc: func(t *testing.T, data any) {
				t.Helper()

				// Use reflection to check the anonymous struct.
				if val := reflect.ValueOf(data); val.Kind() == reflect.Struct {
					categoryField := val.FieldByName("Category")
					if !categoryField.IsValid() || categoryField.String() != "lang" {
						t.Errorf("Expected Category field with 'lang', got %v", categoryField)
					}
				} else {
					t.Errorf("Expected struct, got %T", data)
				}
			},
		},
		{
			name:         "CacheStats returns as-is",
			response:     &CacheStats{TotalFiles: 42, ValidFiles: 30},
			expectedType: "*eol.CacheStats",
			checkFunc: func(t *testing.T, data any) {
				t.Helper()

				if resp, ok := data.(*CacheStats); !ok {
					t.Errorf("Expected *CacheStats, got %T", data)
				} else if resp.TotalFiles != 42 || resp.ValidFiles != 30 {
					t.Errorf("Expected 42/30, got %d/%d", resp.TotalFiles, resp.ValidFiles)
				}
			},
		},
		{
			name:         "Unknown type returns as-is",
			response:     "unknown",
			expectedType: "string",
			checkFunc: func(t *testing.T, data any) {
				t.Helper()

				if str, ok := data.(string); !ok {
					t.Errorf("Expected string, got %T", data)
				} else if str != "unknown" {
					t.Errorf("Expected 'unknown', got %s", str)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := client.extractTemplateData(tt.response)

			if tt.expectedType != "" {
				actualType := reflect.TypeOf(data).String()
				if actualType != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, actualType)
				}
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, data)
			}
		})
	}
}

func TestClientFormatFullProducts(t *testing.T) {
	t.Parallel()

	responses := createMockResponses(t)
	client := createTestClient(t, t.Context(), responses, "products", []string{"--full"})
	products := &FullProductListResponse{
		Result: []ProductDetails{
			{
				Name:     "go",
				Label:    "Go",
				Category: "lang",
				Releases: []ProductRelease{
					{
						Name:         "1.24",
						Label:        "1.24",
						ReleaseDate:  "2025-02-11",
						IsLts:        false,
						IsMaintained: true,
						IsEol:        false,
					},
				},
			},
			{
				Name:     "python",
				Label:    "Python",
				Category: "lang",
				Releases: []ProductRelease{},
			},
		},
	}

	output, err := client.FormatFullProducts(products)
	if err != nil {
		t.Fatalf("FormatFullProducts() error = %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "go") {
		t.Error("Expected output to contain 'go'")
	}

	if !strings.Contains(outputStr, "python") {
		t.Error("Expected output to contain 'python'")
	}

	// Should contain separator for multiple products.
	if !strings.Contains(outputStr, "--------") {
		t.Error("Expected output to contain separator")
	}
}

func TestClientOutputJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data        any
		checkOutput func(string) bool
		name        string
		expectError bool
	}{
		{
			name: "simple object",
			data: map[string]any{"name": "test", "value": 42},
			checkOutput: func(output string) bool {
				return strings.Contains(output, `"name": "test"`) && strings.Contains(output, `"value": 42`)
			},
		},
		{
			name: "ProductResponse",
			data: &ProductResponse{
				Result: ProductDetails{
					Name:     "go",
					Label:    "Go",
					Category: "lang",
				},
			},
			checkOutput: func(output string) bool {
				return strings.Contains(output, `"name": "go"`) && strings.Contains(output, `"category": "lang"`)
			},
		},
		{
			name: "empty object",
			data: map[string]any{},
			checkOutput: func(output string) bool {
				return strings.Contains(output, "{}")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "index", []string{})

			var buf bytes.Buffer

			client.sink = &buf

			err := client.outputJSON(tt.data)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("outputJSON() error = %v", err)
			}

			output := buf.String()
			if tt.checkOutput != nil && !tt.checkOutput(output) {
				t.Errorf("Output check failed. Got: %s", output)
			}
		})
	}
}

func TestClientHandleWithJSONOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		checkJSON   func(string) bool
		name        string
		command     string
		args        []string
		expectError bool
	}{
		{
			name:    "index command with JSON",
			command: "index",
			args:    []string{},
			checkJSON: func(output string) bool {
				return strings.Contains(output, `"schema_version"`) && strings.Contains(output, `"result"`)
			},
		},
		{
			name:    "products command with JSON",
			command: "products",
			args:    []string{},
			checkJSON: func(output string) bool {
				return strings.Contains(output, `"total"`) && strings.Contains(output, `"result"`)
			},
		},
		{
			name:    "product command with JSON",
			command: "product",
			args:    []string{"go"},
			checkJSON: func(output string) bool {
				return strings.Contains(output, `"result"`) && strings.Contains(output, `"last_modified"`)
			},
		},
		{
			name:    "cache stats with JSON",
			command: "cache",
			args:    []string{"stats"},
			checkJSON: func(output string) bool {
				return strings.Contains(output, `"total_files"`) && strings.Contains(output, `"total_size"`)
			},
		},
		{
			name:    "templates with JSON",
			command: "templates",
			args:    []string{},
			checkJSON: func(output string) bool {
				return strings.Contains(output, `"templates"`) && strings.Contains(output, `"total"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, tt.command, tt.args)

			var buf bytes.Buffer

			client.sink = &buf
			client.config.Format = FormatJSON

			err := client.Handle()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}

			output := buf.String()
			if tt.checkJSON != nil && !tt.checkJSON(output) {
				t.Errorf("JSON output check failed. Got: %s", output)
			}
		})
	}
}

func TestClientHandleWithInlineTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		command      string
		template     string
		expectOutput string
		args         []string
		expectError  bool
	}{
		{
			name:         "simple template",
			command:      "index",
			args:         []string{},
			template:     "Total: {{.Total}}",
			expectOutput: "Total: 2",
		},
		{
			name:        "invalid template",
			command:     "index",
			args:        []string{},
			template:    "{{.Invalid",
			expectError: true,
		},
		{
			name:         "template with product data",
			command:      "product",
			args:         []string{"go"},
			template:     "Product: {{.Name}}",
			expectOutput: "Product: go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, tt.command, tt.args)

			var buf bytes.Buffer

			client.sink = &buf
			client.config.InlineTemplate = tt.template

			err := client.Handle()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}

			output := buf.String()
			if tt.expectOutput != "" && !strings.Contains(output, tt.expectOutput) {
				t.Errorf("Expected output to contain %q, got: %s", tt.expectOutput, output)
			}
		})
	}
}

func TestClientPreRouting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		command  string
		expected string
		args     []string
	}{
		{
			name:     "cache with stats",
			command:  "cache",
			args:     []string{"stats"},
			expected: "cache/stats",
		},
		{
			name:     "cache with clear",
			command:  "cache",
			args:     []string{"clear"},
			expected: "cache/clear",
		},
		{
			name:     "cache without args",
			command:  "cache",
			args:     []string{},
			expected: "cache/",
		},
		{
			name:     "templates with export",
			command:  "templates",
			args:     []string{"export", "/tmp"},
			expected: "templates/export",
		},
		{
			name:     "templates without args",
			command:  "templates",
			args:     []string{},
			expected: "templates/list",
		},
		{
			name:     "completion with bash",
			command:  "completion",
			args:     []string{"bash"},
			expected: "completion/bash",
		},
		{
			name:     "completion without args",
			command:  "completion",
			args:     []string{},
			expected: "completion/",
		},
		{
			name:     "other command",
			command:  "products",
			args:     []string{},
			expected: "products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, tt.command, tt.args)

			result := client.preRouting(tt.command)
			if result != tt.expected {
				t.Errorf("preRouting(%q) = %q, expected %q", tt.command, result, tt.expected)
			}
		})
	}
}

func TestClientDetectShell(t *testing.T) {
	tests := []struct {
		name     string
		shellEnv string
		expected string
	}{
		{
			name:     "bash shell",
			shellEnv: "/bin/bash",
			expected: "bash",
		},
		{
			name:     "zsh shell",
			shellEnv: "/usr/bin/zsh",
			expected: "zsh",
		},
		{
			name:     "fish shell defaults to bash",
			shellEnv: "/usr/bin/fish",
			expected: "bash",
		},
		{
			name:     "empty shell defaults to bash",
			shellEnv: "",
			expected: "bash",
		},
		{
			name:     "unknown shell defaults to bash",
			shellEnv: "/usr/bin/unknown",
			expected: "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.shellEnv)

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "completion", []string{})

			result := client.detectShell()
			if result != tt.expected {
				t.Errorf("detectShell() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestClientGenerateCompletionScript(t *testing.T) {
	t.Parallel()

	tests := []struct {
		checkContent func(string) bool
		name         string
		shell        string
	}{
		{
			name:  "bash script",
			shell: "bash",
			checkContent: func(script string) bool {
				return script != "" && strings.Contains(script, "bash")
			},
		},
		{
			name:  "zsh script",
			shell: "zsh",
			checkContent: func(script string) bool {
				return script != "" && strings.Contains(script, "zsh")
			},
		},
		{
			name:  "unknown shell defaults to bash",
			shell: "fish",
			checkContent: func(script string) bool {
				return script != ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, "completion", []string{})

			result := client.generateCompletionScript(tt.shell)
			if tt.checkContent != nil && !tt.checkContent(result) {
				t.Errorf("generateCompletionScript(%q) content check failed", tt.shell)
			}
		})
	}
}

func TestClientHandleEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
	}{
		{
			name:        "handle with response header",
			command:     "product",
			args:        []string{"go"},
			expectError: false,
		},
		{
			name:        "handle cache clear with special case",
			command:     "cache",
			args:        []string{"clear"},
			expectError: false,
		},
		{
			name:        "handle cache unknown subcommand",
			command:     "cache",
			args:        []string{"unknown"},
			expectError: true,
		},
		{
			name:        "handle nil response",
			command:     "index",
			args:        []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			responses := createMockResponses(t)
			client := createTestClient(t, t.Context(), responses, tt.command, tt.args)

			// Special handling for nil response test.
			if tt.name == "handle nil response" {
				// Override the response to nil after calling HandleIndex.
				originalResponse := client.response
				client.response = nil

				defer func() { client.response = originalResponse }()
			}

			err := client.Handle()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
		})
	}
}

func createTestClient(t *testing.T, _ context.Context, responses map[string]*mockResponse, command string, args []string) *Client {
	t.Helper()

	mockClient := newMockClient(responses)
	cacheManager := NewCacheManager(filepath.Join(t.TempDir(), "eol-cache"), DefaultBaseURL, true, time.Hour)
	config := &Config{Command: command, Args: args, Format: FormatText}

	client, err := New(
		WithHTTPClient(mockClient),
		WithCacheManager(cacheManager),
		WithConfig(config),
		WithInitialArgs(append([]string{command}, args...)),
	)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return client
}

func createMockResponses(t *testing.T) map[string]*mockResponse {
	t.Helper()

	return map[string]*mockResponse{
		DefaultBaseURL + "/":                                {Code: http.StatusOK, Body: newIndexResponseBody()},
		DefaultBaseURL + "/products":                        {Code: http.StatusOK, Body: newProductsResponseBody()},
		DefaultBaseURL + "/products/full":                   {Code: http.StatusOK, Body: newProductsResponseBody()},
		DefaultBaseURL + "/products/ubuntu":                 {Code: http.StatusOK, Body: newProductResponseBody()},
		DefaultBaseURL + "/products/go":                     {Code: http.StatusOK, Body: newProductResponseBody()},
		DefaultBaseURL + "/products/ubuntu/releases/latest": {Code: http.StatusOK, Body: newLatestResponseBody()},
		DefaultBaseURL + "/products/go/releases/1.24":       {Code: http.StatusOK, Body: newReleaseResponseBody()},
		DefaultBaseURL + "/categories":                      {Code: http.StatusOK, Body: newCategoriesResponseBody()},
		DefaultBaseURL + "/categories/lang":                 {Code: http.StatusOK, Body: newProductsResponseBody()},
		DefaultBaseURL + "/tags":                            {Code: http.StatusOK, Body: newTagsResponseBody()},
		DefaultBaseURL + "/tags/google":                     {Code: http.StatusOK, Body: newProductsResponseBody()},
		DefaultBaseURL + "/identifiers":                     {Code: http.StatusOK, Body: newIdentifierTypesResponseBody()},
		DefaultBaseURL + "/identifiers/cpe":                 {Code: http.StatusOK, Body: newIdentifiersResponseBody()},
	}
}

func newLatestResponseBody() string {
	return `{
		"schema_version": "1.2.0",
		"result": {
			"name": "22.04",
			"label": "22.04 LTS",
			"releaseDate": "2022-04-21",
			"isLts": true,
			"isMaintained": true,
			"isEol": false,
			"latest": {"name": "22.04.3", "date": "2023-08-10"}
		}
	}`
}

func newReleaseResponseBody() string {
	return `{
		"schema_version": "1.2.0",
		"result": {
			"name": "1.24",
			"label": "1.24",
			"releaseDate": "2025-02-11",
			"isLts": false,
			"isMaintained": true,
			"isEol": false,
			"latest": {"name": "1.24.0", "date": "2025-02-11"}
		}
	}`
}

func newCategoriesResponseBody() string {
	return `{
		"schema_version": "1.2.0",
		"total": 2,
		"result": [
			{"name": "lang", "uri": "` + DefaultBaseURL + `/categories/lang"},
			{"name": "os", "uri": "` + DefaultBaseURL + `/categories/os"}
		]
	}`
}

func newTagsResponseBody() string {
	return `{
		"schema_version": "1.2.0",
		"total": 2,
		"result": [
			{"name": "google", "uri": "` + DefaultBaseURL + `/tags/google"},
			{"name": "lang", "uri": "` + DefaultBaseURL + `/tags/lang"}
		]
	}`
}

func newIdentifierTypesResponseBody() string {
	return `{
		"schema_version": "1.2.0",
		"total": 2,
		"result": [
			{"name": "cpe", "uri": "` + DefaultBaseURL + `/identifiers/cpe/"},
			{"name": "purl", "uri": "` + DefaultBaseURL + `/identifiers/purl/"}
		]
	}`
}

func newIdentifiersResponseBody() string {
	return `{
		"schema_version": "1.2.0",
		"total": 1,
		"result": [
			{"id": "cpe:2.3:a:golang:go", "type": "cpe"}
		]
	}`
}
