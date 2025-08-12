package eol

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestClientIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		expectedSchema string
		statusCode     int
		expectedTotal  int
		expectError    bool
	}{
		{
			name: "successful response",
			mockResponse: `{
				"schema_version": "1.2.0",
				"total": 2,
				"result": [
					{"name": "products", "uri": "` + DefaultBaseURL + `/products"},
					{"name": "categories", "uri": "` + DefaultBaseURL + `/categories"}
				]
			}`,
			statusCode:     200,
			expectError:    false,
			expectedSchema: "1.2.0",
			expectedTotal:  2,
		},
		{
			name:         "server error",
			mockResponse: "Internal Server Error",
			statusCode:   500,
			expectError:  true,
		},
		{
			name:         "not found",
			mockResponse: "Not Found",
			statusCode:   404,
			expectError:  true,
		},
		{
			name:         "invalid json",
			mockResponse: "invalid json",
			statusCode:   200,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockHTTPClient := newMockClient(map[string]*mockResponse{
				DefaultBaseURL + "/": {Code: tt.statusCode, Body: tt.mockResponse},
			})

			client := newTestClientEndpoints(t, mockHTTPClient)
			result, err := client.Index()

			//nolint:nestif // ok
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result.SchemaVersion != tt.expectedSchema {
					t.Errorf("Expected schema_version %s, got %s", tt.expectedSchema, result.SchemaVersion)
				}

				if result.Total != tt.expectedTotal {
					t.Errorf("Expected total %d, got %d", tt.expectedTotal, result.Total)
				}
			}
		})
	}
}

func TestClientProducts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mockResponse  string
		statusCode    int
		expectError   bool
		expectedTotal int
	}{
		{
			name: "successful response",
			mockResponse: `{
				"schema_version": "1.2.0",
				"total": 1,
				"result": [
					{
						"name": "go",
						"label": "Go",
						"category": "lang",
						"uri": "` + DefaultBaseURL + `/products/go",
						"aliases": ["golang"],
						"tags": ["google", "lang"]
					}
				]
			}`,
			statusCode:    200,
			expectError:   false,
			expectedTotal: 1,
		},
		{
			name:         "server error",
			mockResponse: "Internal Server Error",
			statusCode:   500,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockHTTPClient := newMockClient(map[string]*mockResponse{
				DefaultBaseURL + "/products": {Code: tt.statusCode, Body: tt.mockResponse},
			})

			client := newTestClientEndpoints(t, mockHTTPClient)
			result, err := client.Products()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result.Total != tt.expectedTotal {
					t.Errorf("Expected total %d, got %d", tt.expectedTotal, result.Total)
				}
			}
		})
	}
}

func TestClientProductsFull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mockResponse  string
		statusCode    int
		expectError   bool
		expectedTotal int
	}{
		{
			name: "successful response",
			mockResponse: `{
				"schema_version": "1.2.0",
				"total": 1,
				"result": [
					{
						"name": "go",
						"label": "Go",
						"category": "lang",
						"aliases": ["golang"],
						"tags": ["google", "lang"],
						"releases": []
					}
				]
			}`,
			statusCode:    200,
			expectError:   false,
			expectedTotal: 1,
		},
		{
			name:         "server error",
			mockResponse: "Internal Server Error",
			statusCode:   500,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := newMockClient(map[string]*mockResponse{
				DefaultBaseURL + "/products/full": {Code: tt.statusCode, Body: tt.mockResponse},
			})

			client := newClientWithTempCache(t, mockClient)

			// Clear cache to ensure fresh request, especially for /products/full endpoint
			// which is always cached even when caching is disabled.
			client.cacheManager.Clear()

			result, err := client.ProductsFull()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result.Total != tt.expectedTotal {
					t.Errorf("Expected total %d, got %d", tt.expectedTotal, result.Total)
				}
			}
		})
	}
}

func TestClientProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		product      string
		mockResponse string
		expectedName string
		statusCode   int
		expectError  bool
	}{
		{
			name:    "successful response",
			product: "go",
			mockResponse: `{
				"schema_version": "1.2.0",
				"last_modified": "2023-01-01T00:00:00Z",
				"result": {
					"name": "go",
					"label": "Go",
					"category": "lang",
					"aliases": ["golang"],
					"tags": ["google", "lang"],
					"releases": []
				}
			}`,
			statusCode:   200,
			expectError:  false,
			expectedName: "go",
		},
		{
			name:         "empty product name",
			product:      "",
			mockResponse: "",
			statusCode:   200,
			expectError:  true,
		},
		{
			name:         "product not found",
			product:      "nonexistent",
			mockResponse: "Not Found",
			statusCode:   404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mockClient *http.Client
			if tt.product != "" {
				mockClient = newMockClient(map[string]*mockResponse{
					DefaultBaseURL + "/products/" + tt.product: {Code: tt.statusCode, Body: tt.mockResponse},
				})
			} else {
				mockClient = newMockClient(map[string]*mockResponse{})
			}

			client := newTestClientEndpoints(t, mockClient)
			result, err := client.Product(tt.product)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result.Result.Name != tt.expectedName {
					t.Errorf("Expected name %s, got %s", tt.expectedName, result.Result.Name)
				}
			}
		})
	}
}

func TestClientProductRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		product      string
		release      string
		mockResponse string
		expectedName string
		statusCode   int
		expectError  bool
	}{
		{
			name:    "successful response",
			product: "go",
			release: "1.24",
			mockResponse: `{
				"schema_version": "1.2.0",
				"result": {
					"name": "1.24",
					"label": "1.24",
					"releaseDate": "2025-02-11",
					"isLts": false,
					"isMaintained": true,
					"isEol": false
				}
			}`,
			statusCode:   200,
			expectError:  false,
			expectedName: "1.24",
		},
		{
			name:        "empty product name",
			product:     "",
			release:     "1.24",
			expectError: true,
		},
		{
			name:        "empty release name",
			product:     "go",
			release:     "",
			expectError: true,
		},
		{
			name:         "release not found",
			product:      "go",
			release:      "999",
			mockResponse: "Not Found",
			statusCode:   404,
			expectError:  true,
		},
		{
			name:    "semantic version normalization",
			product: "go",
			release: "1.24.6",
			mockResponse: `{
				"schema_version": "1.2.0",
				"result": {
					"name": "1.24",
					"label": "1.24",
					"releaseDate": "2025-02-11",
					"isLts": false,
					"isMaintained": true,
					"isEol": false
				}
			}`,
			statusCode:   200,
			expectError:  false,
			expectedName: "1.24",
		},
		{
			name:    "version with zero patch normalization",
			product: "go",
			release: "1.23.0",
			mockResponse: `{
				"schema_version": "1.2.0",
				"result": {
					"name": "1.23",
					"label": "1.23",
					"releaseDate": "2024-08-13",
					"isLts": false,
					"isMaintained": true,
					"isEol": false
				}
			}`,
			statusCode:   200,
			expectError:  false,
			expectedName: "1.23",
		},
		{
			name:         "server error",
			product:      "go",
			release:      "1.24",
			mockResponse: "Internal Server Error",
			statusCode:   500,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mockClient *http.Client

			if tt.product != "" && tt.release != "" {
				normalizedRelease := normalizeVersion(tt.release)
				url := fmt.Sprintf("%s/products/%s/releases/%s", DefaultBaseURL, tt.product, normalizedRelease)
				mockClient = newMockClient(map[string]*mockResponse{
					url: {Code: tt.statusCode, Body: tt.mockResponse},
				})
			} else {
				mockClient = newMockClient(map[string]*mockResponse{})
			}

			client := newTestClientEndpoints(t, mockClient)
			result, err := client.ProductRelease(tt.product, tt.release)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result.Result.Name != tt.expectedName {
					t.Errorf("Expected name %s, got %s", tt.expectedName, result.Result.Name)
				}
			}
		})
	}
}

func TestClientProductLatestRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		product      string
		mockResponse string
		expectedName string
		statusCode   int
		expectError  bool
	}{
		{
			name:    "successful response",
			product: "go",
			mockResponse: `{
				"schema_version": "1.2.0",
				"result": {
					"name": "1.24",
					"label": "1.24",
					"releaseDate": "2025-02-11",
					"isLts": false,
					"isMaintained": true,
					"isEol": false,
					"latest": {
						"name": "1.24.6",
						"date": "2025-08-06",
						"link": "https://go.dev/doc/devel/release#go1.24.minor"
					}
				}
			}`,
			statusCode:   200,
			expectError:  false,
			expectedName: "1.24",
		},
		{
			name:        "empty product name",
			product:     "",
			expectError: true,
		},
		{
			name:         "product not found",
			product:      "nonexistent",
			mockResponse: "Not Found",
			statusCode:   404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mockClient *http.Client
			if tt.product != "" {
				mockClient = newMockClient(map[string]*mockResponse{
					fmt.Sprintf("%s/products/%s/releases/latest", DefaultBaseURL, tt.product): {Code: tt.statusCode, Body: tt.mockResponse},
				})
			} else {
				mockClient = newMockClient(map[string]*mockResponse{})
			}

			client := newTestClientEndpoints(t, mockClient)
			result, err := client.ProductLatestRelease(tt.product)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result.Result.Name != tt.expectedName {
					t.Errorf("Expected name %s, got %s", tt.expectedName, result.Result.Name)
				}
			}
		})
	}
}

func TestClientCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mockResponse  string
		statusCode    int
		expectError   bool
		expectedTotal int
	}{
		{
			name: "successful response",
			mockResponse: `{
				"schema_version": "1.2.0",
				"total": 2,
				"result": [
					{"name": "lang", "uri": "` + DefaultBaseURL + `/categories/lang"},
					{"name": "os", "uri": "` + DefaultBaseURL + `/categories/os"}
				]
			}`,
			statusCode:    200,
			expectError:   false,
			expectedTotal: 2,
		},
		{
			name:         "server error",
			mockResponse: "Internal Server Error",
			statusCode:   500,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockHTTPClient := newMockClient(map[string]*mockResponse{
				DefaultBaseURL + "/categories": {Code: tt.statusCode, Body: tt.mockResponse},
			})

			client := newTestClientEndpoints(t, mockHTTPClient)
			result, err := client.Categories()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result.Total != tt.expectedTotal {
					t.Errorf("Expected total %d, got %d", tt.expectedTotal, result.Total)
				}
			}
		})
	}
}

func TestClientProductsByCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		category     string
		mockResponse string
		statusCode   int
		expectError  bool
	}{
		{
			name:     "successful response",
			category: "lang",
			mockResponse: `{
				"schema_version": "1.2.0",
				"total": 1,
				"result": [
					{
						"name": "go",
						"label": "Go",
						"category": "lang",
						"uri": "` + DefaultBaseURL + `/products/go",
						"aliases": ["golang"],
						"tags": ["google", "lang"]
					}
				]
			}`,
			statusCode:  200,
			expectError: false,
		},
		{
			name:        "empty category name",
			category:    "",
			expectError: true,
		},
		{
			name:         "category not found",
			category:     "nonexistent",
			mockResponse: "Not Found",
			statusCode:   404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mockClient *http.Client
			if tt.category != "" {
				mockClient = newMockClient(map[string]*mockResponse{
					DefaultBaseURL + "/categories/" + tt.category: {Code: tt.statusCode, Body: tt.mockResponse},
				})
			} else {
				mockClient = newMockClient(map[string]*mockResponse{})
			}

			client := newTestClientEndpoints(t, mockClient)
			_, err := client.ProductsByCategory(tt.category)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestClientTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mockResponse  string
		statusCode    int
		expectError   bool
		expectedTotal int
	}{
		{
			name: "successful response",
			mockResponse: `{
				"schema_version": "1.2.0",
				"total": 2,
				"result": [
					{"name": "google", "uri": "` + DefaultBaseURL + `/tags/google"},
					{"name": "lang", "uri": "` + DefaultBaseURL + `/tags/lang"}
				]
			}`,
			statusCode:    200,
			expectError:   false,
			expectedTotal: 2,
		},
		{
			name:         "server error",
			mockResponse: "Internal Server Error",
			statusCode:   500,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockHTTPClient := newMockClient(map[string]*mockResponse{
				DefaultBaseURL + "/tags": {Code: tt.statusCode, Body: tt.mockResponse},
			})

			client := newTestClientEndpoints(t, mockHTTPClient)
			result, err := client.Tags()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result.Total != tt.expectedTotal {
					t.Errorf("Expected total %d, got %d", tt.expectedTotal, result.Total)
				}
			}
		})
	}
}

func TestClientProductsByTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		tag          string
		mockResponse string
		statusCode   int
		expectError  bool
	}{
		{
			name: "successful response",
			tag:  "google",
			mockResponse: `{
				"schema_version": "1.2.0",
				"total": 1,
				"result": [
					{
						"name": "go",
						"label": "Go",
						"category": "lang",
						"uri": "` + DefaultBaseURL + `/products/go",
						"aliases": ["golang"],
						"tags": ["google", "lang"]
					}
				]
			}`,
			statusCode:  200,
			expectError: false,
		},
		{
			name:        "empty tag name",
			tag:         "",
			expectError: true,
		},
		{
			name:         "tag not found",
			tag:          "nonexistent",
			mockResponse: "Not Found",
			statusCode:   404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mockClient *http.Client
			if tt.tag != "" {
				mockClient = newMockClient(map[string]*mockResponse{
					DefaultBaseURL + "/tags/" + tt.tag: {Code: tt.statusCode, Body: tt.mockResponse},
				})
			} else {
				mockClient = newMockClient(map[string]*mockResponse{})
			}

			client := newTestClientEndpoints(t, mockClient)
			_, err := client.ProductsByTag(tt.tag)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestClientIdentifierTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mockResponse  string
		statusCode    int
		expectError   bool
		expectedTotal int
	}{
		{
			name: "successful response",
			mockResponse: `{
				"schema_version": "1.2.0",
				"total": 2,
				"result": [
					{"name": "cpe", "uri": "` + DefaultBaseURL + `/identifiers/cpe/"},
					{"name": "purl", "uri": "` + DefaultBaseURL + `/identifiers/purl/"}
				]
			}`,
			statusCode:    200,
			expectError:   false,
			expectedTotal: 2,
		},
		{
			name:         "server error",
			mockResponse: "Internal Server Error",
			statusCode:   500,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockHTTPClient := newMockClient(map[string]*mockResponse{
				DefaultBaseURL + "/identifiers": {Code: tt.statusCode, Body: tt.mockResponse},
			})

			client := newTestClientEndpoints(t, mockHTTPClient)
			result, err := client.IdentifierTypes()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result.Total != tt.expectedTotal {
					t.Errorf("Expected total %d, got %d", tt.expectedTotal, result.Total)
				}
			}
		})
	}
}

func TestClientIdentifiersByType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		identifierType string
		mockResponse   string
		statusCode     int
		expectError    bool
	}{
		{
			name:           "successful response",
			identifierType: "cpe",
			mockResponse: `{
				"schema_version": "1.2.0",
				"total": 1,
				"result": [
					{
						"identifier": "cpe:2.3:a:golang:go:*:*:*:*:*:*:*:*",
						"product": {
							"name": "go",
							"uri": "` + DefaultBaseURL + `/products/go"
						}
					}
				]
			}`,
			statusCode:  200,
			expectError: false,
		},
		{
			name:           "empty identifier type",
			identifierType: "",
			expectError:    true,
		},
		{
			name:           "identifier type not found",
			identifierType: "nonexistent",
			mockResponse:   "Not Found",
			statusCode:     404,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mockClient *http.Client
			if tt.identifierType != "" {
				mockClient = newMockClient(map[string]*mockResponse{
					DefaultBaseURL + "/identifiers/" + tt.identifierType: {Code: tt.statusCode, Body: tt.mockResponse},
				})
			} else {
				mockClient = newMockClient(map[string]*mockResponse{})
			}

			client := newTestClientEndpoints(t, mockClient)
			_, err := client.IdentifiersByType(tt.identifierType)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestClientProductReleaseSmartCaching(t *testing.T) {
	t.Parallel()

	// Mock response for product endpoint that includes releases.
	productResponse := `{
		"schema_version": "1.2.0",
		"last_modified": "2025-01-11T00:00:00Z",
		"result": {
			"name": "go",
			"label": "Go",
			"category": "lang",
			"aliases": ["golang"],
			"tags": ["google", "lang"],
			"releases": [
				{
					"name": "1.24",
					"label": "1.24",
					"releaseDate": "2025-02-11",
					"isLts": false,
					"isMaintained": true,
					"isEol": false,
					"latest": {"name": "1.24.0", "date": "2025-02-11"}
				},
				{
					"name": "1.23",
					"label": "1.23",
					"releaseDate": "2024-08-13",
					"isLts": false,
					"isMaintained": true,
					"isEol": false,
					"latest": {"name": "1.23.4", "date": "2024-12-03"}
				}
			]
		}
	}`

	// Create mock client with ONLY the product endpoint - no release endpoint.
	// This proves that the release call uses cached data instead of making API call.
	mockClient := newMockClient(map[string]*mockResponse{
		DefaultBaseURL + "/products/go": {Code: http.StatusOK, Body: productResponse},
		// Intentionally NOT including the release endpoint to verify smart caching.
	})

	client := newClientWithTempCache(t, mockClient)

	// First, call Product to populate the cache.
	productResult, err := client.Product("go")
	if err != nil {
		t.Fatalf("Unexpected error getting product: %v", err)
	}

	if productResult.Result.Name != "go" {
		t.Errorf("Expected product name 'go', got '%s'", productResult.Result.Name)
	}

	// Now call ProductRelease - this should use the cached product data.
	// If it tries to make an API call, it will fail because we didn't mock the release endpoint.
	releaseResult, err := client.ProductRelease("go", "1.24")
	if err != nil {
		t.Fatalf("Unexpected error getting release (should have used cached product data): %v", err)
	}

	if releaseResult.Result.Name != "1.24" {
		t.Errorf("Expected release name '1.24', got '%s'", releaseResult.Result.Name)
	}

	if releaseResult.Result.Label != "1.24" {
		t.Errorf("Expected release label '1.24', got '%s'", releaseResult.Result.Label)
	}

	// Test with version normalization - should also work.
	releaseResult2, err := client.ProductRelease("go", "1.23.4") // This should normalize to "1.23".
	if err != nil {
		t.Fatalf("Unexpected error getting release with normalization: %v", err)
	}

	if releaseResult2.Result.Name != "1.23" {
		t.Errorf("Expected normalized release name '1.23', got '%s'", releaseResult2.Result.Name)
	}
}

func TestClientHigherLevelSmartCaching(t *testing.T) {
	t.Parallel()

	// Mock response for ProductsFull endpoint that includes multiple products with releases.
	productsFullResponse := `{
		"schema_version": "1.2.0",
		"total": 2,
		"result": [
			{
				"name": "go",
				"label": "Go",
				"category": "lang",
				"aliases": ["golang"],
				"tags": ["google", "lang"],
				"releases": [
					{
						"name": "1.24",
						"label": "1.24",
						"releaseDate": "2025-02-11",
						"isLts": false,
						"isMaintained": true,
						"isEol": false,
						"latest": {"name": "1.24.0", "date": "2025-02-11"}
					},
					{
						"name": "1.23",
						"label": "1.23",
						"releaseDate": "2024-08-13",
						"isLts": false,
						"isMaintained": true,
						"isEol": false,
						"latest": {"name": "1.23.4", "date": "2024-12-03"}
					}
				]
			},
			{
				"name": "python",
				"label": "Python",
				"category": "lang",
				"aliases": ["python3"],
				"tags": ["language"],
				"releases": [
					{
						"name": "3.13",
						"label": "3.13",
						"releaseDate": "2024-10-07",
						"isLts": false,
						"isMaintained": true,
						"isEol": false,
						"latest": {"name": "3.13.0", "date": "2024-10-07"}
					}
				]
			}
		]
	}`

	// Create mock client with ONLY the ProductsFull endpoint.
	// This proves that all other calls use cached data.
	mockClient := newMockClient(map[string]*mockResponse{
		DefaultBaseURL + "/products/full": {Code: http.StatusOK, Body: productsFullResponse},
	})

	client := newClientWithTempCache(t, mockClient)

	// Step 1: Call ProductsFull to populate the cache.
	fullResult, err := client.ProductsFull()
	if err != nil {
		t.Fatalf("Unexpected error getting ProductsFull: %v", err)
	}

	if fullResult.Total != 2 {
		t.Errorf("Expected total 2, got %d", fullResult.Total)
	}

	// Step 2: Call Products() - should use cached ProductsFull data.
	productsResult, err := client.Products()
	if err != nil {
		t.Fatalf("Unexpected error getting Products (should use cached full data): %v", err)
	}

	if productsResult.Total != 2 {
		t.Errorf("Expected products total 2, got %d", productsResult.Total)
	}

	expectedNames := []string{"go", "python"}
	for i, product := range productsResult.Result {
		if product.Name != expectedNames[i] {
			t.Errorf("Expected product name '%s', got '%s'", expectedNames[i], product.Name)
		}

		if product.URI == "" {
			t.Error("Expected URI to be populated")
		}
	}

	// Step 3: Call Product("go") - should use cached ProductsFull data.
	productResult, err := client.Product("go")
	if err != nil {
		t.Fatalf("Unexpected error getting Product (should use cached full data): %v", err)
	}

	if productResult.Result.Name != "go" {
		t.Errorf("Expected product name 'go', got '%s'", productResult.Result.Name)
	}

	if len(productResult.Result.Releases) != 2 {
		t.Errorf("Expected 2 releases, got %d", len(productResult.Result.Releases))
	}

	// Step 4: Call ProductRelease("go", "1.24") - should use cached ProductsFull data.
	releaseResult, err := client.ProductRelease("go", "1.24")
	if err != nil {
		t.Fatalf("Unexpected error getting ProductRelease (should use cached full data): %v", err)
	}

	if releaseResult.Result.Name != "1.24" {
		t.Errorf("Expected release name '1.24', got '%s'", releaseResult.Result.Name)
	}

	// Step 5: Call ProductRelease for a different product - should also use cached data.
	releaseResult2, err := client.ProductRelease("python", "3.13")
	if err != nil {
		t.Fatalf("Unexpected error getting ProductRelease for python: %v", err)
	}

	if releaseResult2.Result.Name != "3.13" {
		t.Errorf("Expected release name '3.13', got '%s'", releaseResult2.Result.Name)
	}

	// Step 6: Test version normalization with ProductRelease.
	releaseResult3, err := client.ProductRelease("go", "1.23.4") // Should normalize to "1.23".
	if err != nil {
		t.Fatalf("Unexpected error getting normalized release: %v", err)
	}

	if releaseResult3.Result.Name != "1.23" {
		t.Errorf("Expected normalized release name '1.23', got '%s'", releaseResult3.Result.Name)
	}

	// All the above operations should have used only the initial ProductsFull API call.
	// The mock client would fail if any other endpoints were called.
}

// newTestClientEndpoints creates a client with isolated cache for endpoints testing.
func newTestClientEndpoints(t *testing.T, httpClient *http.Client) *Client {
	t.Helper()

	cacheManager := NewCacheManager(t.TempDir(), DefaultBaseURL, true, time.Hour)
	config := &Config{
		Format: FormatText,
	}

	client, err := New(WithHTTPClient(httpClient), WithCacheManager(cacheManager), WithConfig(config))
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return client
}
