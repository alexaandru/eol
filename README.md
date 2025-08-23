# EndOfLife.date API Client

[![Test](https://github.com/alexaandru/eol/actions/workflows/ci.yml/badge.svg)](https://github.com/alexaandru/eol/actions/workflows/ci.yml)
![Coverage](coverage-badge.svg)
![Go](go-badge.svg)

A Go command-line tool for the [endoflife.date](https://endoflife.date) API v1.

The endoflife.date API provides information about end-of-life dates and support lifecycles
for various products including operating systems, frameworks, databases, and other software products.

## Features

- **Complete API Coverage**: All endoflife.date API v1 endpoints
- **Zero Dependencies**: Uses only Go standard library
- **Template Based**: Customizable output formatting
- **Version Fallback**: Automatic fallback for versions (1.24.6 → 1.24 → 1)
- **JSON Output**: Machine-readable output for automation
- **Shell Completion**: Bash and Zsh completion support
- **Badge Creation**: As a side-kick, it can generate badges for
  releases, color coded appropriately based on EOL ![Red](nokia-c21.svg),
  EOAS ![Orange](ubuntu-22.04.svg) or maintained ![Green](aws-lambda-provided.al2023.svg).

## Installation

### As a CLI Tool

```bash
go install github.com/alexaandru/eol@latest # or
go get -tool github.com/alexaandru/eol
```

## CLI Usage

### Basic Commands

```bash
# Get help and version
eol help
eol version

# List products
eol products
eol products-full                # Detailed info

# Product information
eol product ubuntu
eol release ubuntu 22.04
eol release go 1.24.999          # Will try 1.24.999 → 1.24 → 1
eol latest ubuntu

# Generate SVG badges
eol release-badge go 1.21        # Generate SVG badge with color-coded status
eol release-badge ubuntu 22.04   # Width adjusts to text length automatically

# Browse by category/tag
eol categories                   # List categories
eol category os                  # Products in 'os' category
eol tags                         # List tags
eol tag canonical                # Products with 'canonical' tag

# Identifiers
eol identifiers                  # List identifier types
eol identifier cpe               # CPE identifiers

# Template management
eol templates-export             # Export templates to default location
eol templates-export --templates-dir ~/my-templates  # Export to custom directory
```

### Output Formats

```bash
# JSON output (perfect for scripting)
eol -f json products
eol -f json product ubuntu | jq '.result.releases[0]'

# Custom templates
eol -t '{{.name}}: {{.category}}' product ubuntu
eol -t '{{.name}}' latest go
eol -t '{{if .isMaintained}}✅ Active{{else}}💀 EOL{{end}}' latest terraform

# Generate SVG badges
eol release-badge go 1.21 > go-1.21-badge.svg
eol release-badge ubuntu 22.04 > ubuntu-badge.svg

# Clean lists from collections
eol category os -t '{{join (toStringSlice (collect "name" .)) " "}}'
eol products -t '{{join (toStringSlice (collect "category" .)) ", "}}'

# Scripting with exit codes
eol release go 1.17 -t '{{if .isEol}}{{exit 1}}{{end}}'  # Exit code 1 if EOL
```

### Scripting & Automation

The `exit` template function enables conditional exit codes for shell scripting:

```bash
# Check if a product version is EOL and exit with error code
eol release go 1.17 -t '{{if .isEol}}{{exit 1}}{{end}}'
echo $?  # Will be 1 if EOL, 0 if maintained

# Version fallback in action - tries 1.999 → 1 and checks if EOL
eol release go 1.999 -t '{{if .isEol}}{{exit 1}}{{end}}'

# Use in shell scripts for automated checks
if eol release ubuntu 18.04 -t '{{if .isEol}}{{exit 1}}{{end}}' 2>/dev/null; then
    echo "Ubuntu 18.04 is still supported"
else
    echo "Ubuntu 18.04 is EOL - time to upgrade!"
fi
```

The `eolWithin` function enables proactive EOL monitoring:

```bash
# Check for upcoming EOL within 6 months
eol product nodejs -t '{{range .releases}}{{if eolWithin "6mo" .eolFrom}}⚠️  {{.name}} EOLs {{.eolFrom}}{{"\n"}}{{end}}{{end}}'

# Exit with error if EOL is within 30 days
eol release ubuntu 20.04 -t '{{if eolWithin "30d" .eolFrom}}URGENT: EOL in 30 days!{{exit 2}}{{end}}'

# Monitor multiple releases and exit on first warning
eol product go -t '{{range .releases}}{{if eolWithin "3mo" .eolFrom}}{{.name}} EOLs soon: {{.eolFrom}}{{exit 1}}{{end}}{{end}}'

# Use in CI/CD pipelines for dependency checks
if eol product python -t '{{range .releases}}{{if eolWithin "12mo" .eolFrom}}{{exit 1}}{{end}}{{end}}' 2>/dev/null; then
    echo "Python version will EOL within a year - plan migration"
fi
```

### Template Customization

```bash
# Use custom template directory
eol --templates-dir ~/my-templates product go

# Export templates to a custom directory
eol templates-export --templates-dir ~/my-templates

# Inline templates with various template functions
eol -t '{{join (toStringSlice .tags) ", "}}' product go
eol -t '{{toJSON .}}' product ubuntu
```

### Shell Completion

Enable command-line completion for faster CLI usage:

```bash
# Auto-detect shell and load completion
eval $(eol completion)

# Or specify shell explicitly
eval $(eol completion-bash)
eval $(eol completion-zsh)

# Optional: Save to completion files for permanent installation
eol completion-bash > ~/.bash_completion.d/eol
eol completion-zsh > ~/.zsh/completions/_eol
```

**Benefits of using `eval $(eol completion)`:**

- Always uses the latest completion script matching your binary version
- No need to manually update completion files when commands change
- Completions automatically include new commands like `release-badge` and `templates-export`
- Product names, categories, and tags are fetched dynamically from the API

The completion supports:

- **All commands and subcommands** (including `release-badge`, `templates-export`, `version`)
- **Global flags** (`-f`, `--format`, `--templates-dir`, etc.)
- **Format options** (`text`, `json`)
- **Dynamic product/category/tag completion** from live API data
- **Smart context-aware suggestions** (e.g., version completion for release commands)

### Version Fallback

The tool automatically tries version variants when a specific version isn't found:

```bash
# If 1.24.999 doesn't exist, it will try 1.24, then 1
eol release go 1.24.999

# If 3.11.5 doesn't exist, it will try 3.11, then 3
eol release python 3.11.5

# Only retries on 404 (Not Found) errors - other errors bubble up immediately
```

### Example Output

```bash
$ eol version
eol - EndOfLife.date API client v1.0.0

$ eol product ubuntu
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

Releases 🚀: (42):
  25.04 (25.04 'Plucky Puffin') - Released: 2025-04-17 - EOL: 2026-01-17 - LTS: false - Maintained: true
  24.04 (24.04 'Noble Numbat' (LTS)) - Released: 2024-04-25 - EOL: 2029-04-25 - LTS: true - Maintained: true
  ...

$ eol -f json latest go
{
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

## Template Functions

Available in custom templates:

- `join (toStringSlice .tags) ", "` - Join string slices with separator
- `toJSON .` - Convert to JSON format
- `eolWithin "6mo" .eolFrom` - Check if EOL is within duration (supports mo, wk, d, h, m, s)
- `dict "key1" "value1" "key2" "value2"` - Create a dictionary
- `toStringSlice .field` - Convert to string slice
- `collect "fieldname" .slice` - Extract field from slice of objects for clean joining
- `add .a .b` - Addition (integers)
- `mul .a .b` - Multiplication (integers)
- `exit 1` - Exit with error code (for scripting)

Custom duration formats for `eolWithin`:

- `6mo` - 6 months
- `2wk` - 2 weeks
- `30d` - 30 days
- Also supports standard Go durations: `24h`, `30m`, etc.

### The `collect` Function

The `collect` function solves the common problem of extracting fields from arrays of objects for clean output:

```bash
# Problem: Using range creates unwanted spaces
eol category os -t '{{range .}} {{.name}}{{end}}'
# Output: " almalinux alpine-linux ubuntu " (extra spaces)

# Solution: Use collect to extract fields cleanly
eol category os -t '{{join (toStringSlice (collect "name" .)) " "}}'
# Output: "almalinux alpine-linux ubuntu" (clean)

# More examples:
eol products -t '{{join (toStringSlice (collect "category" .)) ", "}}'  # Categories
eol product ubuntu -t '{{join (toStringSlice (collect "name" .releases)) "\n"}}'  # Versions
```

The `collect` function extracts the specified field from each object in a slice, returning a clean array that works perfectly with `join`.

## Template Directory Structure

Place custom `.tmpl` files in your template directory:

```
~/my-templates/
├── product.tmpl
├── release.tmpl
├── categories.tmpl
└── ...
```

File names should match command names. Use `{{define "templatename"}}...{{end}}` if you want explicit template definitions.

Export built-in templates to get started:

```bash
# Export to default location (~/.config/eol/templates)
eol templates-export

# Export to custom directory
eol templates-export --templates-dir ~/my-templates

# Then use your custom templates
eol --templates-dir ~/my-templates product go
```

## SVG Badges

The `release-badge` command generates SVG badges for releases with automatic color coding:

- **Green** (#4c1): Maintained releases
- **Red** (#e05d44): End-of-life (EOL) releases
- **Orange** (#fe7d37): End-of-active-support (EOAS) releases
- **Gray** (#9f9f9f): Other releases

The badge width automatically adjusts based on text length:

```bash
eol release-badge go 1.21 > go-badge.svg
eol release-badge kubernetes 1.28 > k8s-badge.svg
```

Perfect for README files, documentation, and CI/CD status displays.

## Links

- [endoflife.date website](https://endoflife.date/)
- [API documentation](https://endoflife.date/docs/api/v1/)
- [GitHub repository](https://github.com/endoflife-date/endoflife.date)

## License

[MIT](LICENSE)
