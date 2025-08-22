# EndOfLife.date API Client

[![Test](https://github.com/alexaandru/eol/actions/workflows/ci.yml/badge.svg)](https://github.com/alexaandru/eol/actions/workflows/ci.yml)
![Coverage](coverage-badge.svg)

A Go client library and command-line tool for the [endoflife.date](https://endoflife.date) API v1.

The endoflife.date API provides information about end-of-life dates and support lifecycles for various products including operating systems, frameworks, databases, and other software products.

## Features

- **Complete API Coverage**: All endoflife.date API v1 endpoints
- **Zero Dependencies**: Uses only Go standard library
- **Type Safety**: Full type definitions for all API responses
- **Smart Caching**: File-based caching with configurable TTL
- **Template System**: Customizable output formatting
- **Version Fallback**: Automatic fallback for versions (1.24.6 → 1.24 → 1)
- **JSON Output**: Machine-readable output for automation

## Installation

### As a Library

```bash
go get github.com/alexaandru/eol
```

### As a CLI Tool

```bash
go install github.com/alexaandru/eol/cmd/eol@latest # OR, even better
go get -tool github.com/alexaandru/eol/cmd/eol
```

## Library Usage

### Quick Start

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/alexaandru/eol"
)

func main() {
    client, err := eol.New()
    if err != nil {
        log.Fatal(err)
    }

    // Get all products
    products, err := client.Products()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d products\n", products.Total)

    // Get specific product details
    ubuntu, err := client.Product("ubuntu")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Ubuntu: %s\n", ubuntu.Result.Label)

    // Get specific release with automatic version fallback
    release, err := client.ProductRelease("go", "1.24.999")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Go release: %s\n", release.Result.Name)  // Will try 1.24.999 → 1.24 → 1
}
```

### Configuration Options

```go
client, err := eol.New(
    eol.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
    eol.WithBaseURL("https://custom.api.url"),
)
```

### Available Methods

```go
// Products
client.Products()                           // List all products (summary)
client.ProductsFull()                       // Full details (use sparingly)
client.Product("ubuntu")                    // Specific product
client.ProductRelease("go", "1.24")         // Specific release
client.ProductLatestRelease("ubuntu")       // Latest release

// Categories & Tags
client.Categories()                         // List categories
client.ProductsByCategory("os")             // Products in category
client.Tags()                               // List tags
client.ProductsByTag("google")              // Products with tag

// Identifiers
client.IdentifierTypes()                    // List types (cpe, purl, etc.)
client.IdentifiersByType("cpe")             // Identifiers by type

// Meta
client.Index()                              // API endpoints
```

### Version Fallback

The library automatically tries version variants when a specific version isn't found:

```go
// If 1.24.999 doesn't exist, it will try 1.24, then 1
release, err := client.ProductRelease("go", "1.24.999")

// If 3.11.5 doesn't exist, it will try 3.11, then 3
release, err := client.ProductRelease("python", "3.11.5")

// Only retries on 404 (Not Found) errors - other errors bubble up immediately
```

## CLI Usage

### Basic Commands

```bash
# Get help
eol help

# List products
eol products
eol products --full              # Detailed info (cached 24h)

# Product information
eol product ubuntu
eol release ubuntu 22.04
eol release go 1.24.999          # Will try 1.24.999 → 1.24 → 1
eol latest ubuntu

# Browse by category/tag
eol categories                   # List categories
eol categories os                # Products in 'os' category
eol tags                         # List tags
eol tags canonical               # Products with 'canonical' tag

# Identifiers
eol identifiers                  # List identifier types
eol identifiers cpe              # CPE identifiers
```

### Output Formats

```bash
# JSON output (perfect for scripting)
eol -f json products
eol -f json product ubuntu | jq '.result.releases[0]'

# Custom templates
eol -t '{{.Name}}: {{.Category}}' product ubuntu
eol -t '{{.Latest.Name}}' latest go
eol -t '{{if .IsMaintained}}✅ Active{{else}}💀 EOL{{end}}' latest terraform

# Scripting with exit codes
eol release go 1.17 -t '{{if .IsEol}}{{exit 1}}{{end}}'  # Exit code 1 if EOL
```

### Scripting & Automation

The `exit` template function enables conditional exit codes for shell scripting:

```bash
# Check if a product version is EOL and exit with error code
eol release go 1.17 -t '{{if .IsEol}}{{exit 1}}{{end}}'
echo $?  # Will be 1 if EOL, 0 if maintained

# Version fallback in action - tries 1.999 → 1 and checks if EOL
eol release go 1.999 -t '{{if .IsEol}}{{exit 1}}{{end}}'

# Use in shell scripts for automated checks
if eol release ubuntu 18.04 -t '{{if .IsEol}}{{exit 1}}{{end}}' 2>/dev/null; then
    echo "Ubuntu 18.04 is still supported"
else
    echo "Ubuntu 18.04 is EOL - time to upgrade!"
fi
```

The `eol_within` function enables proactive EOL monitoring:

```bash
# Check for upcoming EOL within 6 months
eol product nodejs -t '{{range .Releases}}{{if eol_within "6mo" .EolFrom}}⚠️  {{.Name}} EOLs {{.EolFrom}}{{"\n"}}{{end}}{{end}}'

# Exit with error if EOL is within 30 days
eol release ubuntu 20.04 -t '{{if eol_within "30d" .EolFrom}}URGENT: EOL in 30 days!{{exit 2}}{{end}}'

# Monitor multiple releases and exit on first warning
eol product go -t '{{range .Releases}}{{if eol_within "3mo" .EolFrom}}{{.Name}} EOLs soon: {{.EolFrom}}{{exit 1}}{{end}}{{end}}'

# Use in CI/CD pipelines for dependency checks
if eol product python -t '{{range .Releases}}{{if eol_within "12mo" .EolFrom}}{{exit 1}}{{end}}{{end}}' 2>/dev/null; then
    echo "Python version will EOL within a year - plan migration"
fi
```

### Caching & Performance

```bash
# Default caching (1 hour)
eol product ubuntu

# Custom cache duration
eol --cache-for 2h product ubuntu
eol --cache-for 30m latest go

# Disable caching
eol --disable-cache product ubuntu

# Cache management
eol cache stats                  # Show cache statistics
eol cache clear                  # Clear all cache

# Shell completion
eol completion                   # Auto-detect shell and generate completion script
eol completion bash              # Generate bash completion script
eol completion zsh               # Generate zsh completion script
```

### Template Customization

```bash
# List available templates
eol templates

# Export templates for customization
eol templates export ~/.config/eol/templates

# Use custom template directory
eol --template-dir ~/my-templates product go

# Shell completion setup
eol completion bash > ~/.bash_completion.d/eol
eol completion zsh > ~/.zsh/completions/_eol
source ~/.bash_completion.d/eol  # For bash
```

### Shell Completion

Enable command-line completion for faster CLI usage:

```bash
# Auto-detect shell and install completion
eol completion > ~/.local/share/bash-completion/completions/eol

# Or specify shell explicitly
eol completion bash > ~/.bash_completion.d/eol
eol completion zsh > ~/.zsh/completions/_eol

# Reload your shell or source the completion file
source ~/.bash_completion.d/eol
```

The completion supports:

- **All commands and subcommands** (`cache stats`, `templates export`, etc.)
- **Global flags** (`-f`, `--format`, `--template-dir`, etc.)
- **Format options** (`text`, `json`)
- **Smart context-aware suggestions**

### Example Output

```bash
$ eol product ubuntu
Product details (last modified: 2025-08-11 00:28:35):

Name: ubuntu
Label: Ubuntu
Category: os
Aliases: ubuntu-linux
Tags: linux-distribution, os
Version Command: cat /etc/os-release
Identifiers:
  cpe: cpe:2.3:o:canonical:ubuntu_linux
  cpe: cpe:/o:canonical:ubuntu_linux
Links:
  HTML: https://endoflife.date/ubuntu
  Icon: https://cdn.jsdelivr.net/npm/simple-icons/icons/ubuntu.svg
  Release Policy: https://wiki.ubuntu.com/Releases
Labels:
  EOL: Maintenance & Security Support
  EOAS: Hardware & Maintenance
  EOES: Expanded Security Maintenance

Releases (42):
  25.04 (25.04 'Plucky Puffin') - Released: 2025-04-17 - EOL: 2026-01-17 - LTS: false - Maintained: true
  24.04 (24.04 'Noble Numbat' (LTS)) - Released: 2024-04-25 - EOL: 2029-04-25 - LTS: true - Maintained: true
  ...

$ eol -f json latest go
{
  "schema_version": "1.2.0",
  "result": {
    "name": "1.24",
    "label": "1.24",
    "releaseDate": "2025-02-11",
    "isLts": false,
    "isEol": false,
    "isMaintained": true,
    "latest": {
      "name": "1.24.6",
      "date": "2025-08-06"
    }
  }
}
```

## Response Structure

All API responses follow this structure:

```go
type Response struct {
    SchemaVersion string `json:"schema_version"`
    Total         int    `json:"total"`         // For list responses
    Result        T      `json:"result"`        // Actual data
}
```

### Product Information

```go
type ProductDetails struct {
    Name           string           `json:"name"`
    Label          string           `json:"label"`
    Category       string           `json:"category"`
    Tags           []string         `json:"tags"`
    Aliases        []string         `json:"aliases"`
    VersionCommand *string          `json:"versionCommand"`
    Identifiers    []Identifier     `json:"identifiers"`
    Links          ProductLinks     `json:"links"`
    Labels         ProductLabels    `json:"labels"`
    Releases       []ProductRelease `json:"releases"`
    // ... more fields
}
```

### Release Information

```go
type ProductRelease struct {
    Name           string  `json:"name"`
    Label          string  `json:"label"`
    ReleaseDate    string  `json:"releaseDate"`
    IsLts          bool    `json:"isLts"`
    IsEol          bool    `json:"isEol"`
    IsMaintained   bool    `json:"isMaintained"`
    Latest         *ProductVersion `json:"latest"`
    // ... more fields
}
```

## Error Handling

The client handles common scenarios gracefully:

```go
product, err := client.Product("nonexistent")
if err != nil {
    // Error includes HTTP status and context
    fmt.Printf("Error: %v\n", err)
}
```

Common error cases:

- `404 Not Found` - Product/resource doesn't exist
- `429 Too Many Requests` - Rate limit exceeded
- Network timeouts and connectivity issues

## Performance & Rate Limiting

**Be considerate of the free API:**

- Use `products` instead of `products --full` when possible
- The CLI includes automatic caching to reduce API load
- `--full` endpoint is limited to 24-hour caching
- Implement retry logic with backoff for production use

**Caching behavior:**

- Default: 1 hour for all endpoints except --full
- `--full`: Always 24 hours (cannot be disabled)
- Location: `~/.cache/eol/` (configurable with `--cache-dir`)
- Format: `.eol_cache.json` files with expiration metadata

**Cache safety:**

The `cache clear` command includes multiple safety layers:

- Only works if the final folder name is exactly: `.eol-cache`, `eol-cache`, or `eol`
- Only removes `*.eol_cache.json` files (no subfolders affected)
- Will conservatively refuse to clear folders that do not look like our own cache folder
- Example safe paths: `~/.cache/eol`, `/var/cache/eol-cache`, `/tmp/.eol-cache`

## Template Functions

Available in custom templates:

- `join .Tags ", "` - Join string slices
- `add .A .B` - Addition
- `sub .A .B` - Subtraction
- `div .A .B` - Division (float64)
- `mul .A .B` - Multiplication (float64)
- `default "fallback" .Field` - Default values
- `toJSON .` - Convert to JSON
- `slice .Releases 0 5` - Slice operations
- `eol_within "6mo" .EolFrom` - Check if EOL is within duration (supports mo, wk, d, h, m, s)
- `exit 1` - Exit with error code (for scripting)

## Testing & Coverage

The project includes comprehensive testing with both unit and integration coverage:

```bash
# Run unit tests with coverage
make test

# Run integration tests with coverage (tests actual binary execution)
make integration-coverage

# Run all tests and checks
make all

# Generate HTML coverage report
go tool cover -html=integration.cov -o coverage.html
```

The integration tests use Go's built-in coverage instrumentation (`go build -cover`) to collect coverage data from actual binary execution, following the approach described in the [Go blog](https://go.dev/blog/integration-test-coverage).

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## Links

- [endoflife.date website](https://endoflife.date/)
- [API documentation](https://endoflife.date/docs/api/v1/)
- [GitHub repository](https://github.com/endoflife-date/endoflife.date)

## License

[MIT](LICENSE)
