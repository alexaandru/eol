package eol

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// OutputFormat represents the output format.
type OutputFormat int

// Config holds the CLI configuration.
type Config struct {
	Command        string
	CacheDir       string
	TemplateDir    string
	InlineTemplate string
	Args           []string
	CacheTTL       time.Duration
	Format         OutputFormat
	CacheEnabled   bool
}

// Supported output formats.
const (
	FormatText OutputFormat = iota
	FormatJSON
)

var (
	errRequires    = errors.New("requires")
	errUnsupported = errors.New("unsupported")
)

// NewConfig creates a new Config with default values.
// If initial arguments are provided, it uses them, otherwise it defaults to os.Args.
func NewConfig(opts ...string) (c *Config, err error) {
	var args []string

	if opts != nil {
		args = opts
	} else {
		args = os.Args[1:]
	}

	if len(args) < 1 {
		return nil, fmt.Errorf("%w: %w a command", ErrUsage, errRequires)
	}

	c = &Config{Format: FormatText, CacheEnabled: true, CacheTTL: DefaultCacheTTL}

	args, err = c.ParseGlobalFlags(args)
	if err != nil {
		return
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("%w: %w an argument", ErrUsage, errRequires)
	}

	c.Command, c.Args = args[0], args[1:]

	return
}

// ParseGlobalFlags parses global command-line flags from the provided arguments
// and returns the remaining non-flag arguments.
//
//nolint:gocognit,gocyclo,cyclop,funlen // ok
func (c *Config) ParseGlobalFlags(args []string) (rest []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-f", "--format":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%w: -f/--format %w a value", ErrUsage, errRequires)
			}

			i++

			format := args[i]
			switch format {
			case "json":
				c.Format = FormatJSON
			case "text":
				c.Format = FormatText
			default:
				return nil, fmt.Errorf("%w format '%s'", errUnsupported, format)
			}
		case "--disable-cache":
			c.CacheEnabled = false
		case "--cache-dir":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%w: --cache-dir %w a directory path", ErrUsage, errRequires)
			}

			i++
			c.CacheDir = args[i]
		case "--cache-for":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%w: --cache-for %w a duration", ErrUsage, errRequires)
			}

			var duration time.Duration

			duration, err = time.ParseDuration(args[i+1])
			if err != nil {
				return nil, err
			}

			c.CacheTTL = duration
			i++ // Skip the duration argument.
		case "--template-dir":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%w: --template-dir %w a directory path", ErrUsage, errRequires)
			}

			c.TemplateDir = args[i+1]
			i++ // Skip the directory argument.
		case "-t", "--template":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%w: --template %w a template string", ErrUsage, errRequires)
			}

			c.InlineTemplate = args[i+1]
			i++ // Skip the template argument.
		default:
			rest = append(rest, arg)
		}
	}

	return
}

// IsJSON returns true if the output format is JSON.
func (c *Config) IsJSON() bool {
	return c.Format == FormatJSON
}

// IsText returns true if the output format is text.
func (c *Config) IsText() bool {
	return c.Format == FormatText
}

// HasInlineTemplate returns true if an inline template is specified.
func (c *Config) HasInlineTemplate() bool {
	return c.InlineTemplate != ""
}
