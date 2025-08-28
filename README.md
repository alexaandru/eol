# EndOfLife.date API Client

[![Test](https://github.com/alexaandru/eol/actions/workflows/ci.yml/badge.svg)](https://github.com/alexaandru/eol/actions/workflows/ci.yml)
![Coverage](coverage-badge.svg)
![Go](go-badge.svg)

A Go command-line tool for the [endoflife.date](https://endoflife.date) [API v1](https://endoflife.date/docs/api/v1/).

The endoflife.date API provides information about end-of-life dates and support lifecycles
for various products including operating systems, frameworks, databases, and other software products.

## Features

- **Complete API Coverage**: All endoflife.date API endpoints;
- **Zero Dependencies**: Uses only Go standard library;
- **Template Based**: Customizable output formatting;
- **Version Fallback**: Automatic fallback for versions (1.24.6 → 1.24 → 1);
- **JSON Output**: Machine-readable output for automation;
- **Shell Completion**: Bash and Zsh completion support;
- **Badge Creation**: As a side-kick, it can generate badges for
  releases, color coded appropriately based on EOL ![Red](nokia-c21.svg),
  EOAS ![Orange](ubuntu-22.04.svg) or maintained ![Green](aws-lambda-provided.al2023.svg).

## Installation

```bash
go install github.com/alexaandru/eol@latest # or
go get -tool github.com/alexaandru/eol
```

You can enable command-line completion:

```bash
eol completion-bash > ~/.bash_completion.d/eol # or, for zsh
eol completion-zsh > ~/.zsh/autoload/_eol && echo "compdef _eol eol" >> ~/.zshrc
```

## Usage

### Commands Overview

```bash
# Get help and version
eol help
eol version

# List products
eol products
eol products-full                # Full info. Preferably use products to get a summary
                                 # and reduce the amount of data transferred.
# Product information
eol product ubuntu
eol release ubuntu 22.04
eol release go 1.24.6            # Will try 1.24.6 → 1.24 → 1
eol latest ubuntu
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
```

Most commands also support the following options:

```bash
# Output format
eol -f json # or text, the default
eol -f json product ubuntu | jq '.result.releases[0]'

# Custom, inline templates
eol -t '{{.name}}: {{.category}}' product ubuntu
eol -t '{{.name}}' latest go
eol -t '{{if .isMaintained}}✅ Active{{else}}💀 EOL{{end}}' latest terraform

# Custom, on disk templates
eol --templates-dir ~/my-templates templates-export # and edit as needed, then
eol --templates-dir ~/my-templates product go
```

### Template Customization

The tool uses Go [text/template](https://pkg.go.dev/text/template@go1.25.0), so
you can leverage all the capabilities of Go templates. Additionally, it provides
several custom template functions to make your life easier:

- `join (toStringSlice .tags) ", "` - Join string slices with separator;
- `toJSON .` - Convert to JSON format;
- `eolWithin "6mo" .eolFrom` - Check if EOL is within duration (supports mo, wk, d, h, m, s),
  enabling proactive monitoring;
- `dict "key1" "value1" "key2" "value2"` - Create a dictionary;
- `toStringSlice .field` - Convert to string slice;
- `collect "fieldname" .slice` - Extract field from slice of objects for clean joining;
- `add .a .b` - Addition (integers);
- `mul .a .b` - Multiplication (integers);
- `exit 1` - Exit with error code (for scripting).

Note that while the cli itself will not exit with error on eol, etc. you can easily
control that via templates by leveraging the `exit` template function, i.e.:

```bash
eol release go 1.17 -t '{{if .isEol}}{{exit 1}}{{else if .isEoas}}{{exit 2}}{{end}}'
echo $?  # Will be 1 if EOL, 2 if EOAS, 0 if maintained
```

### Version Fallback

The tool automatically tries version variants when a specific version isn't found:

```bash
# If 1.24.6 doesn't exist, it will try 1.24, then 1
eol release go 1.24.6

# If 3.11.5 doesn't exist, it will try 3.11, then 3
eol release python 3.11.5
```

## License

[MIT](LICENSE)
