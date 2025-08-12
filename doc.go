// Package eol provides a comprehensive Go client library and command-line tool
// for the endoflife.date API v1.
//
// The endoflife.date API provides information about end-of-life dates and support
// lifecycles for various products including operating systems, frameworks,
// databases, and other software products.
//
// # Features
//
// This package offers:
//   - Complete API coverage for all endoflife.date API v1 endpoints
//   - Dual usage as both Go library and command-line tool
//   - Zero external dependencies (uses only Go standard library)
//   - Full type safety with comprehensive type definitions
//   - Smart file-based caching with configurable TTL
//   - Customizable template system for output formatting
//   - Automatic semantic version normalization
//   - JSON output support for automation and scripting
//   - Shell completion support
//   - Configurable HTTP client with timeout and proxy support
//
// # Quick Start
//
// Create a client and query for products:
//
//	client, err := eol.New()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Get all products (summary)
//	products, err := client.Products()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Found %d products\n", products.Total)
//
//	// Get specific product details
//	ubuntu, err := client.Product("ubuntu")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Ubuntu: %s\n", ubuntu.Result.Label)
//
//	// Get latest release information
//	latest, err := client.ProductLatestRelease("go")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Latest Go: %s\n", latest.Result.Name)
//
// # Configuration Options
//
// The client supports various configuration options using functional options:
//
//	import (
//		"net/http"
//		"time"
//	)
//
//	client, err := eol.New(
//		eol.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
//		eol.WithBaseURL("https://custom.api.url"),
//		eol.WithCacheManager(customCacheManager),
//		eol.WithTemplateManager(customTemplateManager),
//	)
//
// # API Endpoints
//
// The client supports all endoflife.date API v1 endpoints:
//
//   - Index() - List available API endpoints
//   - Products() - List all products (summary view)
//   - ProductsFull() - List all products with full details
//   - Product(name) - Get specific product details
//   - ProductRelease(product, release) - Get specific release information
//   - ProductLatestRelease(product) - Get latest release information
//   - Categories() - List all available categories
//   - ProductsByCategory(category) - Get products in a specific category
//   - Tags() - List all available tags
//   - ProductsByTag(tag) - Get products with a specific tag
//   - IdentifierTypes() - List available identifier types (CPE, PURL, etc.)
//   - IdentifiersByType(type) - Get identifiers by type
//
// # Semantic Version Handling
//
// The library automatically normalizes semantic versions to the format expected by the API.
// Patch versions are automatically stripped to major.minor format:
//
//	// These calls are equivalent - "1.24.6" is normalized to "1.24"
//	release1, err := client.ProductRelease("go", "1.24.6")
//	release2, err := client.ProductRelease("go", "1.24")
//
// # Smart Caching
//
// The client includes intelligent caching to reduce API load and improve performance:
//
//   - Default cache TTL: 1 hour for most endpoints
//   - ProductsFull endpoint: 24-hour cache (cannot be disabled)
//   - Smart cache sharing: Product details can be served from ProductsFull cache
//   - File-based cache storage in ~/.cache/eol/ (follows XDG conventions)
//   - Configurable cache directory and TTL
//   - Cache files use .eol_cache.json extension for clear identification
//
// # Cache Safety
//
// The cache clear operation includes multiple safety layers to prevent accidental data loss:
//
//   - Only works if the final folder name is exactly: .eol-cache, eol-cache, or eol
//   - Only removes *.eol_cache.json files (no subfolders affected)
//   - Will conservatively refuse to clear folders that do not look like our own cache folder
//   - Example safe paths: ~/.cache/eol, /var/cache/eol-cache, /tmp/.eol-cache
//
// # Error Handling
//
// The client handles common HTTP errors gracefully:
//
//	product, err := client.Product("nonexistent")
//	if err != nil {
//		// Error includes HTTP status and helpful context
//		fmt.Printf("Error: %v\n", err) // "Not Found (404)"
//	}
//
// Common error scenarios:
//   - 404 Not Found - Product or resource does not exist
//   - 429 Too Many Requests - Rate limit exceeded
//   - Network timeouts and connectivity issues
//   - JSON parsing errors
//
// # Response Structure
//
// All API responses follow a consistent structure with schema versioning:
//
//	type Response struct {
//		SchemaVersion string `json:"schema_version"`
//		Total         int    `json:"total"`    // For list responses
//		Result        T      `json:"result"`   // The actual data
//	}
//
// # Command Line Usage
//
// This package includes a comprehensive command-line interface. Build and install:
//
//		go install github.com/alexaandru/eol/cmd/eol@latest # OR
//	 go get -tool github.com/alexaandru/eol/cmd/eol
//
// Basic usage examples:
//
//	eol products                    # List all products
//	eol product ubuntu              # Get Ubuntu details
//	eol latest go                   # Get latest Go release
//	eol release go 1.24.6           # Get specific Go release (auto-normalized)
//	eol categories                  # List categories
//	eol categories os               # Products in 'os' category
//	eol tags                        # List tags
//	eol tags google                 # Products with 'google' tag
//
// Output formatting options:
//
//	eol -f json products            # JSON output for scripting
//	eol -t '{{.Name}}: {{.Category}}' product ubuntu  # Custom template
//	eol --cache-for 2h product ubuntu                  # Custom cache duration
//	eol --disable-cache latest go                      # Disable caching
//	eol cache clear                                    # Safely clear cache
//	eol cache stats                                    # Show cache statistics
//
// # Template System
//
// The package includes a powerful template system for custom output formatting:
//
// Available template functions:
//   - join - Join string slices
//   - add, sub, mul, div - Arithmetic operations
//   - default - Provide default values
//   - toJSON - Convert to JSON
//   - slice - Slice operations
//   - exit - Exit with specific code (for scripting)
//
// Example template usage:
//
//	eol -t '{{.Latest.Name}}' latest go
//	eol -t '{{if .IsEol}}💀 EOL{{else}}✅ Active{{end}}' latest terraform
//	eol -t '{{if .IsEol}}{{exit 1}}{{end}}' release ubuntu 18.04  # Exit code for scripting
//
// # Performance Considerations
//
// To be respectful of the free endoflife.date API:
//
//   - Use Products() instead of ProductsFull() when summary data is sufficient
//   - Leverage the built-in caching system
//   - ProductsFull() responses are automatically cached for 24 hours
//   - Implement retry logic with exponential backoff for production use
//   - Consider the rate limits when making bulk requests
//
// # Advanced Usage
//
// Custom configuration and cache management:
//
//	// Custom configuration
//	config, err := eol.NewConfig("--cache-for", "2h", "--format", "json")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	client, err := eol.New(eol.WithConfig(config))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Custom cache manager (final folder name must be exactly eol, eol-cache, or .eol-cache)
//	cacheManager := eol.NewCacheManager("/custom/cache/eol", eol.DefaultBaseURL, true, time.Hour*2)
//	client, err := eol.New(eol.WithCacheManager(cacheManager))
//
// For comprehensive documentation and examples, visit:
//   - API documentation: https://endoflife.date/docs/api/v1/
//   - Project repository: https://github.com/alexaandru/eol
//   - endoflife.date website: https://endoflife.date/
package eol
