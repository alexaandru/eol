package eol

import (
	"encoding/json"
	"testing"
	"time"
)

const null = "null"

func TestUriJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		uri      URI
		expected string
	}{
		{
			name:     "basic uri",
			uri:      URI{Name: "products", URI: "https://example.com/products"},
			expected: `{"name":"products","uri":"https://example.com/products"}`,
		},
		{
			name:     "empty uri",
			uri:      URI{},
			expected: `{"name":"","uri":""}`,
		},
		{
			name:     "uri with special characters",
			uri:      URI{Name: "test-name", URI: "https://example.com/api/v1?param=value"},
			expected: `{"name":"test-name","uri":"https://example.com/api/v1?param=value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test marshaling.
			data, err := json.Marshal(tt.uri)
			if err != nil {
				t.Errorf("Failed to marshal Uri: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected JSON %s, got %s", tt.expected, string(data))
			}

			// Test unmarshaling.
			var unmarshaled URI

			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal Uri: %v", err)
			}

			if unmarshaled != tt.uri {
				t.Errorf("Expected %+v, got %+v", tt.uri, unmarshaled)
			}
		})
	}
}

func TestIdentifierJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier Identifier
		expected   string
	}{
		{
			name:       "basic identifier",
			identifier: Identifier{ID: "test-id", Type: "cpe"},
			expected:   `{"id":"test-id","type":"cpe"}`,
		},
		{
			name:       "empty identifier",
			identifier: Identifier{},
			expected:   `{"id":"","type":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test marshaling.
			data, err := json.Marshal(tt.identifier)
			if err != nil {
				t.Errorf("Failed to marshal Identifier: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected JSON %s, got %s", tt.expected, string(data))
			}

			// Test unmarshaling.
			var unmarshaled Identifier

			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal Identifier: %v", err)
			}

			if unmarshaled != tt.identifier {
				t.Errorf("Expected %+v, got %+v", tt.identifier, unmarshaled)
			}
		})
	}
}

func TestProductVersionJSON(t *testing.T) {
	t.Parallel()

	dateStr := "2023-01-01"
	linkStr := "https://example.com"

	tests := []struct {
		name           string
		productVersion ProductVersion
		expectedJSON   string
	}{
		{
			name: "complete product version",
			productVersion: ProductVersion{
				Date: &dateStr,
				Link: &linkStr,
				Name: "1.0.0",
			},
			expectedJSON: `{"date":"2023-01-01","link":"https://example.com","name":"1.0.0"}`,
		},
		{
			name: "minimal product version",
			productVersion: ProductVersion{
				Name: "1.0.0",
			},
			expectedJSON: `{"date":null,"link":null,"name":"1.0.0"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test marshaling.
			data, err := json.Marshal(tt.productVersion)
			if err != nil {
				t.Errorf("Failed to marshal ProductVersion: %v", err)
			}

			if string(data) != tt.expectedJSON {
				t.Errorf("Expected JSON %s, got %s", tt.expectedJSON, string(data))
			}

			// Test unmarshaling.
			var unmarshaled ProductVersion

			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal ProductVersion: %v", err)
			}

			// Compare fields individually due to pointer comparison issues.
			if tt.productVersion.Date != nil && unmarshaled.Date != nil {
				if *tt.productVersion.Date != *unmarshaled.Date {
					t.Errorf("Date mismatch: expected %s, got %s", *tt.productVersion.Date, *unmarshaled.Date)
				}
			} else if tt.productVersion.Date != unmarshaled.Date {
				t.Errorf("Date pointer mismatch")
			}

			if tt.productVersion.Link != nil && unmarshaled.Link != nil {
				if *tt.productVersion.Link != *unmarshaled.Link {
					t.Errorf("Link mismatch: expected %s, got %s", *tt.productVersion.Link, *unmarshaled.Link)
				}
			} else if tt.productVersion.Link != unmarshaled.Link {
				t.Errorf("Link pointer mismatch")
			}

			if tt.productVersion.Name != unmarshaled.Name {
				t.Errorf("Name mismatch: expected %s, got %s", tt.productVersion.Name, unmarshaled.Name)
			}
		})
	}
}

func TestProductReleaseJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		productRelease ProductRelease
		expectJSON     bool
	}{
		{
			name: "complete product release",
			productRelease: ProductRelease{
				Name:         "1.0.0",
				Label:        "1.0.0",
				ReleaseDate:  "2023-01-01",
				IsLts:        true,
				IsMaintained: true,
				IsEol:        false,
			},
			expectJSON: true,
		},
		{
			name: "minimal product release",
			productRelease: ProductRelease{
				Name:        "1.0.0",
				Label:       "1.0.0",
				ReleaseDate: "2023-01-01",
			},
			expectJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test JSON marshaling (should use MarshalJSON).
			data, err := json.Marshal(tt.productRelease)
			if err != nil {
				t.Errorf("Failed to marshal ProductRelease: %v", err)
			}

			// Should be valid JSON.
			var result map[string]any

			err = json.Unmarshal(data, &result)
			if err != nil {
				t.Errorf("Marshaled data is not valid JSON: %v", err)
			}

			// Check that key fields are present.
			if result["name"] != tt.productRelease.Name {
				t.Errorf("Expected name %s, got %v", tt.productRelease.Name, result["name"])
			}

			if result["label"] != tt.productRelease.Label {
				t.Errorf("Expected label %s, got %v", tt.productRelease.Label, result["label"])
			}
		})
	}
}

func TestProductSummaryJSON(t *testing.T) {
	t.Parallel()

	productSummary := ProductSummary{
		Name:     "go",
		Label:    "Go",
		Category: "lang",
		URI:      "https://example.com/go",
		Aliases:  []string{"golang"},
		Tags:     []string{"google", "lang"},
	}

	// Test marshaling.
	data, err := json.Marshal(productSummary)
	if err != nil {
		t.Errorf("Failed to marshal ProductSummary: %v", err)
	}

	// Test unmarshaling.
	var unmarshaled ProductSummary

	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal ProductSummary: %v", err)
	}

	// Compare fields.
	if unmarshaled.Name != productSummary.Name {
		t.Errorf("Name mismatch: expected %s, got %s", productSummary.Name, unmarshaled.Name)
	}

	if len(unmarshaled.Aliases) != len(productSummary.Aliases) {
		t.Errorf("Aliases length mismatch: expected %d, got %d", len(productSummary.Aliases), len(unmarshaled.Aliases))
	}

	if len(unmarshaled.Tags) != len(productSummary.Tags) {
		t.Errorf("Tags length mismatch: expected %d, got %d", len(productSummary.Tags), len(unmarshaled.Tags))
	}
}

func TestProductLabelsJSON(t *testing.T) {
	t.Parallel()

	eoasStr := "End of Active Support"
	discontinuedStr := "Discontinued"
	eoesStr := "End of Extended Support"

	tests := []struct {
		name   string
		labels ProductLabels
	}{
		{
			name: "complete labels",
			labels: ProductLabels{
				Eoas:         &eoasStr,
				Discontinued: &discontinuedStr,
				Eoes:         &eoesStr,
				Eol:          "End of Life",
			},
		},
		{
			name: "minimal labels",
			labels: ProductLabels{
				Eol: "End of Life",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test marshaling.
			data, err := json.Marshal(tt.labels)
			if err != nil {
				t.Errorf("Failed to marshal ProductLabels: %v", err)
			}

			// Test unmarshaling.
			var unmarshaled ProductLabels

			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal ProductLabels: %v", err)
			}

			if unmarshaled.Eol != tt.labels.Eol {
				t.Errorf("Eol mismatch: expected %s, got %s", tt.labels.Eol, unmarshaled.Eol)
			}
		})
	}
}

func TestProductLinksJSON(t *testing.T) {
	t.Parallel()

	iconStr := "https://example.com/icon.png"
	policyStr := "https://example.com/policy"

	productLinks := ProductLinks{
		Icon:          &iconStr,
		ReleasePolicy: &policyStr,
		HTML:          "https://example.com",
	}

	// Test marshaling.
	data, err := json.Marshal(productLinks)
	if err != nil {
		t.Errorf("Failed to marshal ProductLinks: %v", err)
	}

	// Test unmarshaling.
	var unmarshaled ProductLinks

	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal ProductLinks: %v", err)
	}

	if unmarshaled.HTML != productLinks.HTML {
		t.Errorf("HTML mismatch: expected %s, got %s", productLinks.HTML, unmarshaled.HTML)
	}
}

func TestProductDetailsJSON(t *testing.T) {
	t.Parallel()

	productDetails := ProductDetails{
		Name:     "go",
		Label:    "Go",
		Aliases:  []string{"golang"},
		Category: "lang",
		Tags:     []string{"google", "lang"},
		Labels: ProductLabels{
			Eol: "End of Life",
		},
		Links: ProductLinks{
			HTML: "https://example.com",
		},
		Releases: []ProductRelease{
			{
				Name:        "1.0.0",
				Label:       "1.0.0",
				ReleaseDate: "2023-01-01",
			},
		},
	}

	// Test JSON marshaling (should use MarshalJSON).
	data, err := json.Marshal(productDetails)
	if err != nil {
		t.Errorf("Failed to marshal ProductDetails: %v", err)
	}

	// Should be valid JSON.
	var result map[string]any

	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Errorf("Marshaled data is not valid JSON: %v", err)
	}

	// Check that key fields are present.
	if result["name"] != productDetails.Name {
		t.Errorf("Expected name %s, got %v", productDetails.Name, result["name"])
	}

	if result["category"] != productDetails.Category {
		t.Errorf("Expected category %s, got %v", productDetails.Category, result["category"])
	}
}

func TestUriListResponseJSON(t *testing.T) {
	t.Parallel()

	response := URIListResponse{
		SchemaVersion: "1.2.0",
		Total:         2,
		Result: []URI{
			{Name: "products", URI: "https://example.com/products"},
			{Name: "categories", URI: "https://example.com/categories"},
		},
	}

	// Test marshaling.
	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal UriListResponse: %v", err)
	}

	// Test unmarshaling.
	var unmarshaled URIListResponse

	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal UriListResponse: %v", err)
	}

	if unmarshaled.SchemaVersion != response.SchemaVersion {
		t.Errorf("SchemaVersion mismatch: expected %s, got %s", response.SchemaVersion, unmarshaled.SchemaVersion)
	}

	if unmarshaled.Total != response.Total {
		t.Errorf("Total mismatch: expected %d, got %d", response.Total, unmarshaled.Total)
	}

	if len(unmarshaled.Result) != len(response.Result) {
		t.Errorf("Result length mismatch: expected %d, got %d", len(response.Result), len(unmarshaled.Result))
	}
}

func TestProductListResponseJSON(t *testing.T) {
	t.Parallel()

	response := ProductListResponse{
		SchemaVersion: "1.2.0",
		Total:         1,
		Result: []ProductSummary{
			{
				Name:     "go",
				Label:    "Go",
				Category: "lang",
				URI:      "https://example.com/go",
			},
		},
	}

	// Test marshaling.
	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal ProductListResponse: %v", err)
	}

	// Test unmarshaling.
	var unmarshaled ProductListResponse

	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal ProductListResponse: %v", err)
	}

	if unmarshaled.Total != response.Total {
		t.Errorf("Total mismatch: expected %d, got %d", response.Total, unmarshaled.Total)
	}
}

func TestProductResponseJSON(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	response := ProductResponse{
		SchemaVersion: "1.2.0",
		LastModified:  timestamp,
		Result: ProductDetails{
			Name:     "go",
			Label:    "Go",
			Category: "lang",
		},
	}

	// Test marshaling.
	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal ProductResponse: %v", err)
	}

	// Test unmarshaling.
	var unmarshaled ProductResponse

	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal ProductResponse: %v", err)
	}

	if unmarshaled.SchemaVersion != response.SchemaVersion {
		t.Errorf("SchemaVersion mismatch: expected %s, got %s", response.SchemaVersion, unmarshaled.SchemaVersion)
	}
}

func TestProductReleaseResponseJSON(t *testing.T) {
	t.Parallel()

	response := ProductReleaseResponse{
		SchemaVersion: "1.2.0",
		Result: ProductRelease{
			Name:        "1.0.0",
			Label:       "1.0.0",
			ReleaseDate: "2023-01-01",
		},
	}

	// Test marshaling.
	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal ProductReleaseResponse: %v", err)
	}

	// Test unmarshaling.
	var unmarshaled ProductReleaseResponse

	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal ProductReleaseResponse: %v", err)
	}

	if unmarshaled.SchemaVersion != response.SchemaVersion {
		t.Errorf("SchemaVersion mismatch: expected %s, got %s", response.SchemaVersion, unmarshaled.SchemaVersion)
	}
}

func TestIdentifierProductJSON(t *testing.T) {
	t.Parallel()

	identifierProduct := IdentifierProduct{
		Identifier: "test-identifier",
		Product:    URI{Name: "go", URI: "https://example.com/go"},
	}

	// Test marshaling.
	data, err := json.Marshal(identifierProduct)
	if err != nil {
		t.Errorf("Failed to marshal IdentifierProduct: %v", err)
	}

	// Test unmarshaling.
	var unmarshaled IdentifierProduct

	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal IdentifierProduct: %v", err)
	}

	if unmarshaled.Identifier != identifierProduct.Identifier {
		t.Errorf("Identifier mismatch: expected %s, got %s", identifierProduct.Identifier, unmarshaled.Identifier)
	}

	if unmarshaled.Product.Name != identifierProduct.Product.Name {
		t.Errorf("Product name mismatch: expected %s, got %s", identifierProduct.Product.Name, unmarshaled.Product.Name)
	}
}

func TestIdentifierListResponseJSON(t *testing.T) {
	t.Parallel()

	response := IdentifierListResponse{
		SchemaVersion: "1.2.0",
		Total:         1,
		Result: []IdentifierProduct{
			{
				Identifier: "test-id",
				Product:    URI{Name: "go", URI: "https://example.com/go"},
			},
		},
	}

	// Test marshaling.
	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal IdentifierListResponse: %v", err)
	}

	// Test unmarshaling.
	var unmarshaled IdentifierListResponse

	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal IdentifierListResponse: %v", err)
	}

	if unmarshaled.Total != response.Total {
		t.Errorf("Total mismatch: expected %d, got %d", response.Total, unmarshaled.Total)
	}
}

func TestEmptyAndNilFields(t *testing.T) {
	t.Parallel()

	// Test that structures handle empty and nil fields correctly.
	//nolint:govet // ok
	tests := []struct {
		name string
		data any
	}{
		{"empty Uri", URI{}},
		{"empty Identifier", Identifier{}},
		{"empty ProductVersion", ProductVersion{}},
		{"empty ProductRelease", ProductRelease{}},
		{"empty ProductSummary", ProductSummary{}},
		{"empty ProductLabels", ProductLabels{}},
		{"empty ProductLinks", ProductLinks{}},
		{"empty ProductDetails", ProductDetails{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Should be able to marshal to JSON.
			data, err := json.Marshal(tt.data)
			if err != nil {
				t.Errorf("Failed to marshal %s: %v", tt.name, err)
			}

			// Should be valid JSON.
			var result any
			if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
				t.Errorf("Marshaled %s is not valid JSON: %v", tt.name, jsonErr)
			}
		})
	}
}
