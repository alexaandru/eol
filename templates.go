package eol

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

// TemplateManager manages loading and parsing of templates.
type TemplateManager struct {
	templates   map[string]*template.Template
	sources     map[string]string // For debugging: "builtin", "override", "inline".
	funcMap     template.FuncMap
	overrideDir string // Directory to look for user-defined templates.
}

// TemplateInfo represents template metadata for displaying available templates.
type TemplateInfo struct {
	Name        string
	Description string
}

const (
	dirPerm  = 0o750
	filePerm = 0o640
)

//go:embed templates/*.tmpl
var embeddedTemplates embed.FS

// ErrNoOverrideDir is returned when no template override directory is configured.
var ErrNoOverrideDir = errors.New("no override directory configured")

// NewTemplateManager creates a new template manager with eagerly loaded templates.
// If inlineTemplate is provided, it will override the template inferred from command and args.
func NewTemplateManager(overrideDir, inlineTemplate, command string, args []string) (tm *TemplateManager, err error) {
	tm = &TemplateManager{
		overrideDir: overrideDir,
		funcMap:     getTemplateFuncMap(),
		templates:   make(map[string]*template.Template),
		sources:     make(map[string]string),
	}

	targetTemplateName := ""
	if inlineTemplate != "" {
		targetTemplateName = getTemplateNameForCommand(command, args)
	}

	err = tm.prepareTemplates(inlineTemplate, targetTemplateName)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare templates: %w", err)
	}

	return
}

// GetTemplateSource returns the source of a template for debugging.
func (tm *TemplateManager) GetTemplateSource(name string) string {
	return tm.sources[name]
}

// Execute executes a template using the prepared templates.
func (tm *TemplateManager) Execute(name string, data any) ([]byte, error) {
	tmpl := tm.templates[name]
	if tmpl == nil {
		return nil, fmt.Errorf("template %s not found", name) //nolint:err113 // TODO
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.Bytes(), nil
}

// GetAvailableTemplates returns a list of available template names.
func (tm *TemplateManager) GetAvailableTemplates() []string {
	return slices.Collect(maps.Keys(tm.templates))
}

// GetEmbeddedTemplateContent returns the content of an embedded template.
func GetEmbeddedTemplateContent(path string) ([]byte, error) {
	return embeddedTemplates.ReadFile(path)
}

// ExecuteInlineTemplate executes an inline template string with the same function map.
func ExecuteInlineTemplate(templateStr string, data any) (_ []byte, err error) {
	tmpl, err := template.New("inline").Funcs(getTemplateFuncMap()).Parse(templateStr)
	if err != nil {
		return
	}

	buf := bytes.Buffer{}
	if execErr := tmpl.Execute(&buf, data); execErr != nil {
		return nil, execErr
	}

	return buf.Bytes(), nil
}

// ListTemplates returns a list of available templates with their descriptions.
func (tm *TemplateManager) ListTemplates() []TemplateInfo {
	return []TemplateInfo{
		{Name: "cache_stats", Description: "Cache statistics display template"},
		{Name: "categories", Description: "Categories list display template"},
		{Name: "identifiers", Description: "Identifier types list display template"},
		{Name: "identifiers_by_type", Description: "Identifiers by type display template"},
		{Name: "index", Description: "API endpoints list display template"},
		{Name: "product_details", Description: "Product details display template"},
		{Name: "product_release", Description: "Product release display template"},
		{Name: "products", Description: "Products list display template"},
		{Name: "products_by_category", Description: "Products by category display template"},
		{Name: "products_by_tag", Description: "Products by tag display template"},
		{Name: "tags", Description: "Tags list display template"},
		{Name: "template_export", Description: "Template export result display template"},
		{Name: "templates", Description: "Templates list display template"},
	}
}

// ExportTemplates exports all embedded templates to the specified directory.
func (tm *TemplateManager) ExportTemplates(outputDir string) (err error) {
	if err = os.MkdirAll(outputDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	for _, name := range tm.GetAvailableTemplates() {
		var (
			sourcePath = "templates/" + name + ".tmpl"
			targetPath = filepath.Join(outputDir, name+".tmpl")
			content    []byte
		)

		content, err = embeddedTemplates.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", name, err)
		}

		if writeErr := os.WriteFile(targetPath, content, filePerm); writeErr != nil {
			return fmt.Errorf("failed to write template %s: %w", targetPath, writeErr)
		}
	}

	return
}

// ExecuteInline executes an inline template string with the template manager's function map.
func (tm *TemplateManager) ExecuteInline(templateStr string, data any) (_ []byte, err error) {
	tmpl, err := template.New("inline").Funcs(tm.funcMap).Parse(templateStr)
	if err != nil {
		return
	}

	buf := bytes.Buffer{}
	if execErr := tmpl.Execute(&buf, data); execErr != nil {
		return nil, execErr
	}

	return buf.Bytes(), nil
}

// getTemplateFuncMap returns the standard function map used by all templates.
//
//nolint:gocognit,funlen // ok
func getTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"join": strings.Join,
		"toJSON": func(v any) string {
			b, err := json.MarshalIndent(v, "  ", "  ")
			if err != nil {
				return fmt.Sprintf("error: %v", err)
			}

			return string(b)
		},
		"slice": func(s any, start, end int) any {
			switch v := s.(type) {
			case []ProductRelease:
				if start < 0 || start >= len(v) {
					return v[:0]
				}
				if end > len(v) {
					end = len(v)
				}
				if end <= start {
					return v[:0]
				}

				return v[start:end]
			default:
				return s
			}
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}

			return a / b
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"default": func(def, val any) any {
			if val == nil {
				return def
			}
			switch v := val.(type) {
			case string:
				if v == "" {
					return def
				}
			case *string:
				if v == nil || *v == "" {
					return def
				}
			}

			return val
		},
		"exit": func(code int) string {
			os.Exit(code)
			return ""
		},
	}
}

//nolint:gocognit,gocyclo,cyclop // ok
func getTemplateNameForCommand(command string, args []string) string {
	switch command {
	case "index":
		return "index"
	case "products":
		if len(args) > 0 && args[0] == "full" {
			return "product_details"
		}

		return "products"
	case "product":
		return "product_details"
	case "release":
		return "product_release"
	case "latest":
		return "product_release"
	case "categories":
		if len(args) > 0 {
			return "products_by_category"
		}

		return "categories"
	case "tags":
		if len(args) > 0 {
			return "products_by_tag"
		}

		return "tags"
	case "identifiers":
		if len(args) > 0 {
			return "identifiers_by_type"
		}

		return "identifiers"
	case "cache":
		if len(args) > 0 && args[0] == "stats" {
			return "cache_stats"
		}

		return ""
	case "templates":
		if len(args) > 0 && args[0] == "export" {
			return "template_export"
		}

		return "templates"
	default:
		return ""
	}
}

func (tm *TemplateManager) prepareTemplates(inlineTemplate, targetTemplateName string) (err error) {
	if err = tm.loadBuiltinTemplates(); err != nil {
		return fmt.Errorf("failed to load builtin templates: %w", err)
	}

	if tm.overrideDir != "" {
		if err = tm.loadOverrideTemplates(); err != nil {
			return fmt.Errorf("failed to load override templates: %w", err)
		}
	}

	if inlineTemplate != "" && targetTemplateName != "" {
		var tmpl *template.Template

		tmpl, err = template.New(targetTemplateName).Funcs(tm.funcMap).Parse(inlineTemplate)
		if err != nil {
			return fmt.Errorf("failed to parse inline template: %w", err)
		}

		tm.templates[targetTemplateName] = tmpl
		tm.sources[targetTemplateName] = "inline"
	}

	return
}

func (tm *TemplateManager) loadBuiltinTemplates() (err error) {
	entries, err := embeddedTemplates.ReadDir("templates")
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		var (
			name = strings.TrimSuffix(entry.Name(), ".tmpl")
			tmpl *template.Template
		)

		if tmpl, err = tm.loadFromEmbed(name); err != nil {
			return fmt.Errorf("failed to load builtin template %s: %w", name, err)
		}

		tm.templates[name] = tmpl
		tm.sources[name] = "builtin"
	}

	return
}

func (tm *TemplateManager) loadOverrideTemplates() (err error) {
	root, err := os.OpenRoot(tm.overrideDir)
	if err != nil {
		return fmt.Errorf("failed to open override directory %s: %w", tm.overrideDir, err)
	}
	defer root.Close() //nolint:errcheck // ok

	dir, err := root.Open(".")
	if err != nil {
		return fmt.Errorf("failed to open override directory: %w", err)
	}
	defer dir.Close() //nolint:errcheck // ok

	dirEntries, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("failed to read override directory: %w", err)
	}

	for _, entry := range dirEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		var (
			name = strings.TrimSuffix(entry.Name(), ".tmpl")
			tmpl *template.Template
		)

		if tmpl, err = tm.loadFromFile(name); err != nil {
			return fmt.Errorf("failed to load override template %s: %w", name, err)
		}

		tm.templates[name] = tmpl
		tm.sources[name] = "override"
	}

	return
}

func (tm *TemplateManager) loadFromFile(name string) (_ *template.Template, err error) {
	if tm.overrideDir == "" {
		return nil, ErrNoOverrideDir
	}

	root, err := os.OpenRoot(tm.overrideDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open template directory: %w", err)
	}

	defer root.Close() //nolint:errcheck // ok

	file, err := root.Open(name + ".tmpl")
	if err != nil {
		return
	}

	defer file.Close() //nolint:errcheck // ok

	content, err := io.ReadAll(file)
	if err != nil {
		return
	}

	return template.New(name).Funcs(tm.funcMap).Parse(string(content))
}

func (tm *TemplateManager) loadFromEmbed(name string) (_ *template.Template, err error) {
	content, err := embeddedTemplates.ReadFile("templates/" + name + ".tmpl")
	if err != nil {
		return
	}

	return template.New(name).Funcs(tm.funcMap).Parse(string(content))
}
