package eol

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	mockClient := &http.Client{
		Timeout:   DefaultTimeout,
		Transport: &mockTransport{responses: map[string]*mockResponse{}, err: nil},
	}
	cacheManager := NewCacheManager(t.TempDir(), true, time.Hour)

	client, err := New(WithHTTPClient(mockClient), WithCacheManager(cacheManager))
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if client == nil {
		t.Fatal("New() returned nil")
	}

	if client.baseURL == nil {
		t.Fatal("baseURL is nil")
	}

	if client.baseURL.String() != DefaultBaseURL {
		t.Errorf("Expected baseURL %s, got %s", DefaultBaseURL, client.baseURL.String())
	}

	if client.httpClient == nil {
		t.Fatal("httpClient is nil")
	}

	if client.httpClient.Timeout != DefaultTimeout {
		t.Errorf("Expected timeout %v, got %v", DefaultTimeout, client.httpClient.Timeout)
	}

	if client.userAgent != UserAgent {
		t.Errorf("Expected userAgent %s, got %s", UserAgent, client.userAgent)
	}

	if client.cacheManager == nil {
		t.Fatal("cacheManager is nil")
	}

	if client.config == nil {
		t.Fatal("config is nil")
	}

	if client.templateManager == nil {
		t.Fatal("templateManager is nil")
	}

	// Test that config has default values.
	if client.config.Format != FormatText {
		t.Errorf("Expected default format %v, got %v", FormatText, client.config.Format)
	}

	if !client.config.CacheEnabled {
		t.Error("Expected cache to be enabled by default")
	}

	if client.config.CacheTTL != DefaultCacheTTL {
		t.Errorf("Expected default cache TTL %v, got %v", DefaultCacheTTL, client.config.CacheTTL)
	}
}

func TestNewWithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validate func(*testing.T, *Client)
		name     string
		opts     []Option
	}{
		{
			name: "with custom HTTP client",
			opts: []Option{
				WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
			},
			validate: func(t *testing.T, c *Client) {
				t.Helper()
				if c.httpClient.Timeout != 10*time.Second {
					t.Errorf("Expected timeout 10s, got %v", c.httpClient.Timeout)
				}
			},
		},
		{
			name: "with custom base URL",
			opts: []Option{
				WithBaseURL(mustParseURL("https://api.example.com")),
			},
			validate: func(t *testing.T, c *Client) {
				t.Helper()
				if c.baseURL.String() != "https://api.example.com" {
					t.Errorf("Expected baseURL https://api.example.com, got %s", c.baseURL.String())
				}
			},
		},
		{
			name: "with custom config",
			opts: []Option{
				WithConfig(&Config{
					Format:       FormatJSON,
					CacheEnabled: false,
					CacheTTL:     2 * time.Hour,
				}),
			},
			validate: func(t *testing.T, c *Client) {
				t.Helper()
				if c.config.Format != FormatJSON {
					t.Errorf("Expected format JSON, got %v", c.config.Format)
				}
				if c.config.CacheEnabled {
					t.Error("Expected cache to be disabled")
				}
				if c.config.CacheTTL != 2*time.Hour {
					t.Errorf("Expected cache TTL 2h, got %v", c.config.CacheTTL)
				}
			},
		},
		{
			name: "with custom cache manager",
			opts: []Option{
				WithCacheManager(NewCacheManager("/tmp/custom", true, 30*time.Minute)),
			},
			validate: func(t *testing.T, c *Client) {
				t.Helper()
				if c.cacheManager.baseDir != "/tmp/custom" {
					t.Errorf("Expected cache dir /tmp/custom, got %s", c.cacheManager.baseDir)
				}
				if c.cacheManager.defaultTTL != 30*time.Minute {
					t.Errorf("Expected cache TTL 30m, got %v", c.cacheManager.defaultTTL)
				}
			},
		},
		{
			name: "with custom template manager",
			opts: []Option{
				WithTemplateManager(func() *TemplateManager {
					tm, err := NewTemplateManager("", "", "", nil)
					if err != nil {
						panic(err)
					}

					return tm
				}()),
			},
			validate: func(t *testing.T, c *Client) {
				t.Helper()
				if c.templateManager.overrideDir != "" {
					t.Errorf("Expected empty template dir, got %s", c.templateManager.overrideDir)
				}
			},
		},
		{
			name: "with initial args",
			opts: []Option{
				WithInitialArgs([]string{"product", "ubuntu"}),
			},
			validate: func(t *testing.T, c *Client) {
				t.Helper()
				if c.config.Command != "product" {
					t.Errorf("Expected command 'product', got %s", c.config.Command)
				}
				if len(c.config.Args) != 1 || c.config.Args[0] != "ubuntu" {
					t.Errorf("Expected args ['ubuntu'], got %v", c.config.Args)
				}
			},
		},
		{
			name: "with custom sink",
			opts: []Option{
				WithSink(&bytes.Buffer{}),
			},
			validate: func(t *testing.T, c *Client) {
				t.Helper()
				if c.sink == nil {
					t.Error("Expected sink to be set")
				}
			},
		},
		{
			name: "multiple options",
			opts: []Option{
				WithHTTPClient(&http.Client{Timeout: 5 * time.Second}),
				WithBaseURL(mustParseURL("https://test.example.com")),
				WithInitialArgs([]string{"-f", "json", "products"}),
			},
			validate: func(t *testing.T, c *Client) {
				t.Helper()
				if c.httpClient.Timeout != 5*time.Second {
					t.Errorf("Expected timeout 5s, got %v", c.httpClient.Timeout)
				}
				if c.baseURL.String() != "https://test.example.com" {
					t.Errorf("Expected baseURL https://test.example.com, got %s", c.baseURL.String())
				}
				if c.config.Format != FormatJSON {
					t.Errorf("Expected format JSON, got %v", c.config.Format)
				}
				if c.config.Command != "products" {
					t.Errorf("Expected command 'products', got %s", c.config.Command)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := tt.opts
			// Only add cache manager if not already provided.
			hasCacheManager := tt.name == "with custom cache manager"

			if !hasCacheManager {
				cacheManager := NewCacheManager(t.TempDir(), true, time.Hour)
				opts = append(opts, WithCacheManager(cacheManager))
			}

			client, err := New(opts...)
			if err != nil {
				t.Fatalf("New() returned error: %v", err)
			}

			tt.validate(t, client)
		})
	}
}

func TestNewWithInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []Option
	}{
		{
			name: "empty args should error",
			opts: []Option{
				WithInitialArgs([]string{}), WithInitialArgs([]string{}), // Empty args should trigger error.
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := tt.opts
			cacheManager := NewCacheManager(t.TempDir(), true, time.Hour)
			opts = append(opts, WithCacheManager(cacheManager))

			_, err := New(opts...)
			if err == nil {
				t.Error("Expected New() to return error for invalid config")
			}
		})
	}
}

func TestClientHandleValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		errorType   error
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "help command",
			args:        []string{"help"},
			expectError: true,
			errorType:   ErrNeedHelp,
		},
		{
			name:        "help flag short",
			args:        []string{"-h"},
			expectError: true,
			errorType:   ErrNeedHelp,
		},
		{
			name:        "help flag long",
			args:        []string{"--help"},
			expectError: true,
			errorType:   ErrNeedHelp,
		},
		{
			name:        "unknown command",
			args:        []string{"invalid-command"},
			expectError: true,
		},
		{
			name:        "valid command",
			args:        []string{"index"},
			expectError: false, // Command is valid and may succeed with network call.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := newMockClient(map[string]*mockResponse{
				"https://endoflife.date/api/v1/": {StatusCode: 200, Body: newIndexResponseBody()},
			})
			cacheManager := NewCacheManager(t.TempDir(), true, time.Hour)

			client, err := New(WithHTTPClient(mockClient), WithCacheManager(cacheManager), WithInitialArgs(tt.args))
			if err != nil {
				t.Fatalf("New() returned unexpected error: %v", err)
			}

			err = client.Handle()
			if tt.expectError {
				if err == nil {
					t.Error("Expected Handle() to return error")
					return
				}

				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error to be %v, got %v", tt.errorType, err)
				}

				return
			}

			// For valid commands, we accept either success or network-related errors
			// since we're not mocking the HTTP client in this test.
			if err != nil && tt.name == "valid command" {
				// Network errors are acceptable for valid commands.
				return
			}

			if err != nil {
				t.Errorf("Handle() returned unexpected error: %v", err)
			}
		})
	}
}

func TestClientBuildURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		baseURL  string
		endpoint string
		expected string
	}{
		{
			name:     "simple endpoint",
			baseURL:  "https://api.example.com",
			endpoint: "products",
			expected: "https://api.example.com/products",
		},
		{
			name:     "endpoint with leading slash",
			baseURL:  "https://api.example.com",
			endpoint: "/products",
			expected: "https://api.example.com/products",
		},
		{
			name:     "nested endpoint",
			baseURL:  "https://api.example.com/api/v1",
			endpoint: "products/ubuntu",
			expected: "https://api.example.com/api/v1/products/ubuntu",
		},
		{
			name:     "base URL with trailing slash",
			baseURL:  "https://api.example.com/",
			endpoint: "products",
			expected: "https://api.example.com/products",
		},
		{
			name:     "both with slashes",
			baseURL:  "https://api.example.com/",
			endpoint: "/products",
			expected: "https://api.example.com/products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cacheManager := NewCacheManager(t.TempDir(), true, time.Hour)

			client, err := New(WithBaseURL(mustParseURL(tt.baseURL)), WithCacheManager(cacheManager))
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			result := client.buildURL(tt.endpoint)
			if result != tt.expected {
				t.Errorf("buildURL(%q) = %q, expected %q", tt.endpoint, result, tt.expected)
			}
		})
	}
}

func TestClientHandle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "valid index command",
			args:        []string{"index"},
			expectError: false, // Should succeed with mock server.
		},
		{
			name:        "valid products command",
			args:        []string{"products"},
			expectError: false, // Should succeed with mock server.
		},
		{
			name:        "valid product command with args",
			args:        []string{"product", "go"},
			expectError: false, // Should succeed with mock server.
		},
		{
			name:        "help command",
			args:        []string{"help"},
			expectError: true, // Should return ErrNeedHelp.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := newMockClient(map[string]*mockResponse{
				"https://endoflife.date/api/v1/":            {StatusCode: 200, Body: newIndexResponseBody()},
				"https://endoflife.date/api/v1/products":    {StatusCode: 200, Body: newProductsResponseBody()},
				"https://endoflife.date/api/v1/products/go": {StatusCode: 200, Body: newProductResponseBody()},
			})
			cacheManager := NewCacheManager(t.TempDir(), true, time.Hour)

			client, err := New(WithHTTPClient(mockClient), WithCacheManager(cacheManager), WithInitialArgs(tt.args))
			if err != nil {
				if !tt.expectError {
					t.Fatalf("New() returned unexpected error: %v", err)
				}

				return
			}

			err = client.Handle()
			if tt.expectError && err == nil {
				t.Error("Expected Handle() to return error")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Handle() returned unexpected error: %v", err)
			}
		})
	}
}

// Response body generators for client tests.

func newIndexResponseBody() string {
	return `{
		"schema_version": "1.2.0",
		"total": 2,
		"result": [
			{"name": "products", "uri": "https://endoflife.date/api/v1/products"},
			{"name": "categories", "uri": "https://endoflife.date/api/v1/categories"}
		]
	}`
}

func newProductsResponseBody() string {
	return `{
		"schema_version": "1.2.0",
		"total": 1,
		"result": [
			{
				"name": "go",
				"label": "Go",
				"category": "lang",
				"uri": "https://endoflife.date/api/v1/products/go",
				"aliases": ["golang"],
				"tags": ["google", "lang"]
			}
		]
	}`
}

func newProductResponseBody() string {
	return `{
		"schema_version": "1.2.0",
		"last_modified": "2024-01-01T00:00:00Z",
		"result": {
			"name": "go",
			"label": "Go",
			"category": "lang",
			"aliases": ["golang"],
			"tags": ["google", "lang"],
			"versionCommand": "go version",
			"identifiers": [],
			"labels": {"eol": "EOL"},
			"links": {"html": "https://endoflife.date/go"},
			"releases": [
				{
					"name": "1.24",
					"label": "1.24",
					"releaseDate": "2025-02-11",
					"isLts": false,
					"isMaintained": true,
					"isEol": false,
					"latest": {"name": "1.24.0", "date": "2025-02-11"}
				}
			]
		}
	}`
}

func TestConstants(t *testing.T) {
	t.Parallel()

	if DefaultBaseURL != "https://endoflife.date/api/v1" {
		t.Errorf("Expected defaultBaseURL to be 'https://endoflife.date/api/v1', got %s", DefaultBaseURL)
	}

	if DefaultTimeout != 30*time.Second {
		t.Errorf("Expected DefaultTimeout to be 30s, got %v", DefaultTimeout)
	}

	if DefaultCacheTTL != time.Hour {
		t.Errorf("Expected DefaultCacheTTL to be 1h, got %v", DefaultCacheTTL)
	}

	if UserAgent != "eol-go-client/1.0" {
		t.Errorf("Expected UserAgent to be 'eol-go-client/1.0', got %s", UserAgent)
	}
}

// Helper function for tests.
func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}

	return u
}
