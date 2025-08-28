package main

import (
	"bytes"
	"cmp"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/alexaandru/eol/templates"
)

type client struct {
	sink           io.Writer
	response       []byte
	baseURL        *url.URL
	httpClient     //nolint:embeddedstructfieldcheck // nope
	templates      *template.Template
	command        string
	templatesDir   string
	inlineTemplate string
	args           []string
	format         outputFormat
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type outputFormat int

// Default values.
const (
	DefaultTimeout = 30 * time.Second
	DefaultBaseURL = "https://endoflife.date/api/v1"
)

// Supported output formats.
const (
	FormatText outputFormat = iota
	FormatJSON
)

//nolint:gochecknoglobals // ok
var (
	funcMap = template.FuncMap{
		"join": strings.Join, "toJSON": toJSON, "eolWithin": eolWithin, "dict": dict,
		"exit": func(code int) string { os.Exit(code); return "" },
		"add":  func(a, b int) int { return a + b }, "mul": func(a, b int) int { return a * b },
		"collect": collect, "toStringSlice": toStringSlice,
	}
	rawOutput   = []string{"help", "version", "completion", "completion-bash", "completion-zsh", "templates-export"}
	reCustomDur = regexp.MustCompile(`^(\d+)(d|wk|mo)$`)
	userAgent   = "eol-go-client"
	version     = "unk"
)

var (
	errUsage    = errors.New("usage error")
	errNotFound = errors.New("not found")

	// Usage errors.
	errUnknownCommand    = fmt.Errorf("%w: unknown command", errUsage)
	errUnsupportedFormat = fmt.Errorf("%w: unsupported format", errUsage)

	// Operational errors.
	errReleaseNotFound = errors.New("failed to find release for product")
	errInlineTemplate  = errors.New("inline template seems wrong, did you indend to use -f json?")
	errInvalidDuration = errors.New("invalid duration")
	errInvalidDict     = errors.New("invalid dict")
)

//go:embed completions/bash.sh
var bashCompletionScript string

//go:embed completions/zsh.sh
var zshCompletionScript string

func newClient(args []string) (c *client, err error) {
	baseURL, err := url.Parse(DefaultBaseURL)
	if err != nil {
		return
	}

	c = &client{
		sink:    os.Stdout,
		baseURL: baseURL,
		format:  FormatText,
	}

	if err = c.parseFlags(args); err != nil {
		return
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: DefaultTimeout}
	}

	if c.templates == nil { //nolint:nestif // ok
		c.templates = template.New("master").Funcs(funcMap)

		if err = loadTemplates(c.templates, templates.Templates); err != nil {
			return
		}

		if x := c.templatesDir; x != "" && c.command != "templates-export" {
			if err = loadTemplates(c.templates, os.DirFS(x).(fs.ReadDirFS)); err != nil { //nolint:errcheck,forcetypeassert // ok
				return
			}
		}

		if x := c.inlineTemplate; x != "" {
			_, err = c.templates.Parse(fmt.Sprintf(`{{define "_inline"}}%s{{end}}`, x))
		}
	}

	return
}

//nolint:gocyclo,cyclop,funlen // ok
func (c *client) handle() (err error) {
	c.response = nil
	cmd := c.command

	switch cmd {
	case "help":
		c.printHelp()
	case "version":
		c.printVersion()
	case "index":
		err = c.doRequest("/")
	case "products":
		err = c.doRequest("/products")
	case "products-full":
		err = c.doRequest("/products/full")
	case "product":
		err = c.doRequest("/products/" + c.args[0])
	case "release", "release-badge":
		pn, rel := c.args[0], c.args[1]
		versions, found := generateVersionVariants(rel), false

		for _, version := range versions {
			err = c.doRequest("/products/" + pn + "/releases/" + version)
			if err == nil {
				found = true
				break
			}
		}

		if !found {
			err = fmt.Errorf("%w %s with any of the attempted versions: %v",
				errReleaseNotFound, pn, versions)
		}
	case "latest":
		c.command = "release"
		err = c.doRequest("/products/" + c.args[0] + "/releases/latest")
	case "categories":
		err = c.doRequest("/categories")
	case "category":
		err = c.doRequest("/categories/" + c.args[0])
	case "tags":
		err = c.doRequest("/tags")
	case "tag":
		err = c.doRequest("/tags/" + c.args[0])
	case "identifiers":
		err = c.doRequest("/identifiers")
	case "identifier":
		err = c.doRequest("/identifiers/" + c.args[0])
	case "templates-export":
		err = c.templatesExport(c.templatesDir)
	case "completion-bash":
		c.response = []byte(bashCompletionScript)
	case "completion-zsh":
		c.response = []byte(zshCompletionScript)
	default:
		err = fmt.Errorf("%w: %s", errUnknownCommand, c.command)
	}

	if err != nil || c.response == nil {
		return
	}

	if c.format == FormatJSON || slices.Contains(rawOutput, c.command) {
		_, err = c.sink.Write(c.response)
	} else {
		err = c.executeTemplate(c.command)
	}

	return
}

func (c *client) printHelp() {
	c.printHeader()
	c.sink.Write([]byte("\n\n")) //nolint:errcheck,gosec // ok
	c.printUsage()
}

//nolint:errcheck,gosec // ok
func (c *client) printHeader() { c.sink.Write([]byte("eol - EndOfLife.date API client")) }

//nolint:errcheck,gosec // ok
func (c *client) printUsage() { c.sink.Write([]byte(helpText)) }

func (c *client) printVersion() {
	c.printHeader()
	c.sink.Write([]byte(" " + version + "\n")) //nolint:errcheck,gosec // ok
}

//nolint:gocognit,gocyclo,cyclop,funlen,nakedret // ok
func (c *client) parseFlags(args []string) (err error) {
	if len(args) == 0 {
		return fmt.Errorf("%w: requires a command", errUsage)
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-f", "--format":
			if i+1 >= len(args) {
				return fmt.Errorf("%w: -f/--format requires a value", errUsage)
			}

			i++

			format := args[i]
			switch format {
			case "json":
				c.format = FormatJSON
			case "text":
				c.format = FormatText
			default:
				return fmt.Errorf("%w '%s'", errUnsupportedFormat, format)
			}
		case "--templates-dir":
			if i+1 >= len(args) {
				return fmt.Errorf("%w: --templates-dir requires a directory path", errUsage)
			}

			c.templatesDir = args[i+1]
			i++ // Skip the directory argument.
		case "-t", "--template":
			if i+1 >= len(args) {
				return fmt.Errorf("%w: --template requires a template string", errUsage)
			}

			c.inlineTemplate = args[i+1]
			if c.inlineTemplate == "json" {
				return errInlineTemplate
			}

			i++ // Skip the template argument.
		case "-h", "--help", "help":
			c.command = "help"
		default:
			if c.command == "" && !strings.HasPrefix(arg, "-") {
				c.command = arg
			} else {
				c.args = append(c.args, arg)
			}
		}
	}

	if c.command == "" {
		return fmt.Errorf("%w: requires a command", errUsage)
	}

	switch c.command {
	case "completion":
		if shell := os.Getenv("SHELL"); strings.Contains(shell, "zsh") {
			c.command = "completion-zsh"
		} else {
			c.command = "completion-bash"
		}
	case "product", "category", "tag", "identifier", "latest":
		if len(c.args) < 1 || c.args[0] == "" {
			return fmt.Errorf("%w: %s command requires an argument", errUsage, c.command)
		}
	case "release", "release-badge":
		if len(c.args) < 2 || c.args[0] == "" || c.args[1] == "" {
			return fmt.Errorf("%w: %s command requires two arguments", errUsage, c.command)
		}
	}

	return
}

// Executes a template using the prepared templates.
// Inline template is executed via "_inline" name.
func (c *client) executeTemplate(name string) (err error) {
	if c.inlineTemplate != "" {
		name = "_inline"
	}

	tmpl := c.templates.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("template %s %w", name, errNotFound)
	}

	x := map[string]any{}
	if err = json.Unmarshal(c.response, &x); err != nil {
		return
	}

	//nolint:wrapcheck // ok
	switch v := x["result"].(type) {
	case []any:
		return tmpl.Execute(c.sink, v)
	case map[string]any:
		for i, x := range c.args {
			v[fmt.Sprintf("arg%d", i+1)] = x
		}

		return tmpl.Execute(c.sink, v)
	default:
		return tmpl.Execute(c.sink, v)
	}
}

func loadTemplates(t *template.Template, src fs.ReadDirFS) (err error) {
	entries, err := src.ReadDir(".")
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		var (
			f       fs.File
			content []byte
		)

		if f, err = src.Open(entry.Name()); err != nil {
			return
		}

		if content, err = io.ReadAll(f); err != nil {
			return
		}

		name := strings.TrimSuffix(entry.Name(), ".tmpl")
		if !bytes.HasPrefix(content, []byte("{{define")) &&
			!bytes.HasPrefix(content, []byte("{{ define")) &&
			!bytes.HasPrefix(content, []byte("{{- define")) {
			content = fmt.Appendf(nil, `{{define "%s"}}%s{{end}}`, name, content)
		}

		if _, err = t.Parse(string(content)); err != nil {
			return
		}
	}

	return
}

func (c *client) templatesExport(dir string) (err error) {
	dir = cmp.Or(dir, configDir("templates"))
	if err = os.MkdirAll(dir, 0o750); err != nil { //nolint:mnd // ok
		return
	}

	entries, err := templates.Templates.ReadDir(".")
	if err != nil {
		return
	}

	for _, entry := range entries {
		dst := filepath.Join(dir, entry.Name())
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		var (
			f       fs.File
			content []byte
		)

		if f, err = templates.Templates.Open(entry.Name()); err != nil {
			return
		}

		if content, err = io.ReadAll(f); err != nil {
			return
		}

		if err = os.WriteFile(dst, content, 0o640); err != nil { //nolint:mnd // ok
			return
		}
	}

	c.response = fmt.Appendf(nil, "Templates exported to %s", dir)

	return
}

func (c *client) doRequest(endpoint string) (err error) {
	urL := buildURL(*c.baseURL, endpoint)

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, urL, http.NoBody)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", userAgent+"/"+version)
	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close() //nolint:errcheck // ok

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return errNotFound
		}

		return fmt.Errorf("%s (%d)", http.StatusText(resp.StatusCode), resp.StatusCode) //nolint:err113 // ok
	}

	c.response, err = io.ReadAll(resp.Body)

	return
}

//nolint:gochecknoinits // ok
func init() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		version = info.Main.Version
	}
}
