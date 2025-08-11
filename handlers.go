package eol

import (
	"cmp"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
)

// CategoryProductsResponse represents a response containing products filtered by category.
type CategoryProductsResponse struct {
	*ProductListResponse
	Category string
}

// TagProductsResponse represents a response containing products filtered by tag.
type TagProductsResponse struct {
	*ProductListResponse
	Tag string
}

// TypeIdentifiersResponse represents a response containing identifiers filtered by type.
type TypeIdentifiersResponse struct {
	*IdentifierListResponse
	Type string
}

// IndexResponse represents the API index response with available endpoints.
type IndexResponse struct {
	*UriListResponse
}

// CategoriesResponse represents a response containing available categories.
type CategoriesResponse struct {
	*UriListResponse
}

// TagsResponse represents a response containing available tags.
type TagsResponse struct {
	*UriListResponse
}

// IdentifierTypesResponse represents a response containing available identifier types.
type IdentifierTypesResponse struct {
	*UriListResponse
}

// TemplateListResponse represents a response containing available templates.
type TemplateListResponse struct {
	Templates []TemplateInfo `json:"templates"`
	Total     int            `json:"total"`
}

// TemplateExportResponse represents a response from template export operations.
type TemplateExportResponse struct {
	OutputDir string `json:"output_dir"`
	Message   string `json:"message"`
}

// CompletionResponse represents a response containing shell completion scripts.
type CompletionResponse struct {
	Shell  string `json:"shell"`
	Script string `json:"script"`
}

var (
	// ErrNeedHelp indicates that help was requested by the user.
	ErrNeedHelp = errors.New("help requested")

	errProductReleaseNameRequired = errors.New("product name and release name required")
	errProductNameRequired        = errors.New("product name is required")
	errCacheSubcommandRequired    = errors.New("cache subcommand is required (stats, clear)")
	errOutputDirectoryRequired    = errors.New("output directory is required")
	errUnknownResponseType        = errors.New("unknown response type")
	errUnknownCommand             = errors.New("unknown command")
)

//go:embed completions/bash.sh
var bashCompletionScript string

//go:embed completions/zsh.sh
var zshCompletionScript string

// Handle represents the main entry point for handling commands.
//
//nolint:gocyclo,cyclop,funlen // ok
func (c *Client) Handle() (err error) {
	c.response = nil
	c.responseHeader = ""

	switch cmd := c.preRouting(c.config.Command); cmd {
	case "index":
		err = c.HandleIndex()
	case "products":
		err = c.HandleProducts()
	case "product":
		err = c.HandleProduct()
	case "release":
		err = c.HandleRelease()
	case "latest":
		err = c.HandleLatest()
	case "categories":
		err = c.HandleCategories()
	case "tags":
		err = c.HandleTags()
	case "identifiers":
		err = c.HandleIdentifiers()
	case "cache/stats":
		err = c.HandleCacheStats()
	case "cache/clear":
		err = c.HandleCacheClear()
	case "templates/list":
		err = c.HandleTemplates()
	case "templates/export":
		err = c.HandleTemplateExport()
	case "completion/bash":
		err = c.HandleCompletionBash()
	case "completion/zsh":
		err = c.HandleCompletionZsh()
	case "completion/":
		err = c.HandleCompletionAuto()
	case "help", "-h", "--help":
		return ErrNeedHelp
	case "cache/":
		return errCacheSubcommandRequired
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}

	if err != nil {
		return
	}

	if c.response == nil {
		return
	}

	if c.config.HasInlineTemplate() {
		return c.executeInlineTemplate(c.response)
	}

	if c.config.IsJSON() {
		return c.outputJSON(c.response)
	}

	if c.responseHeader != "" {
		c.Printf("%s\n\n", c.responseHeader)
	}

	text, err := c.Format(c.response)
	if err != nil {
		return
	}

	c.Print(string(text))

	return
}

// Format formats the given response according to the client's configuration settings.
//
//nolint:gocyclo,cyclop // ok
func (c *Client) Format(response any) ([]byte, error) {
	switch resp := response.(type) {
	case *IndexResponse:
		return c.templateManager.Execute("index", c.extractTemplateData(resp))
	case *CategoriesResponse:
		return c.templateManager.Execute("categories", c.extractTemplateData(resp))
	case *TagsResponse:
		return c.templateManager.Execute("tags", c.extractTemplateData(resp))
	case *IdentifierTypesResponse:
		return c.templateManager.Execute("identifiers", c.extractTemplateData(resp))
	case *ProductListResponse:
		return c.templateManager.Execute("products", c.extractTemplateData(resp))
	case *FullProductListResponse:
		return c.FormatFullProducts(resp)
	case *ProductResponse:
		return c.templateManager.Execute("product_details", c.extractTemplateData(resp))
	case *ProductReleaseResponse:
		return c.templateManager.Execute("product_release", c.extractTemplateData(resp))
	case *CategoryProductsResponse:
		return c.templateManager.Execute("products_by_category", c.extractTemplateData(resp))
	case *TagProductsResponse:
		return c.templateManager.Execute("products_by_tag", c.extractTemplateData(resp))
	case *TypeIdentifiersResponse:
		return c.templateManager.Execute("identifiers_by_type", c.extractTemplateData(resp))
	case *CacheStats:
		return c.templateManager.Execute("cache_stats", c.extractTemplateData(resp))
	case *TemplateListResponse:
		return c.templateManager.Execute("templates", c.extractTemplateData(resp))
	case *TemplateExportResponse:
		return c.templateManager.Execute("template_export", c.extractTemplateData(resp))
	case *CompletionResponse:
		return []byte(resp.Script), nil
	default:
		return nil, fmt.Errorf("%w: %T", errUnknownResponseType, resp)
	}
}

// FormatFullProducts formats full product list with individual product details.
func (c *Client) FormatFullProducts(products *FullProductListResponse) (result []byte, err error) {
	separator := []byte(strings.Repeat("-", 80) + "\n") //nolint:mnd // separator

	for i := range products.Result {
		var text []byte

		text, err = c.templateManager.Execute("product_details", &products.Result[i])
		if err != nil {
			return nil, fmt.Errorf("error formatting product details: %w", err)
		}

		result = append(result, text...)
		result = append(result, '\n')

		if i < len(products.Result)-1 {
			result = append(result, separator...)
		}
	}

	return
}

// HandleIndex handles the index command.
func (c *Client) HandleIndex() (err error) {
	response, err := c.Index()
	if err != nil {
		return fmt.Errorf("failed to get index: %w", err)
	}

	c.response = &IndexResponse{UriListResponse: response}
	c.responseHeader = ""

	return
}

// HandleProducts handles the products command.
func (c *Client) HandleProducts() (err error) {
	args := c.config.Args
	full := len(args) > 0 && args[0] == "--full"

	if full {
		var r *FullProductListResponse

		if r, err = c.ProductsFull(); err != nil {
			return fmt.Errorf("failed to get full products: %w", err)
		}

		c.response = r
		c.responseHeader = fmt.Sprintf("All products (full details) - %d total:", r.Total)
	} else {
		var r *ProductListResponse

		if r, err = c.Products(); err != nil {
			return fmt.Errorf("failed to get products: %w", err)
		}

		c.response = r
	}

	return
}

// HandleProduct handles the product command.
func (c *Client) HandleProduct() (err error) {
	args := c.config.Args
	if len(args) == 0 {
		return errProductNameRequired
	}

	productName := args[0]
	if productName == "" {
		return errProductNameRequired
	}

	response, err := c.Product(productName)
	if err != nil {
		return fmt.Errorf("failed to get product %s: %w", productName, err)
	}

	c.response = response
	c.responseHeader = fmt.Sprintf("Product details (last modified: %s):",
		response.LastModified.Format("2006-01-02 15:04:05"))

	return
}

// HandleRelease handles the release command.
func (c *Client) HandleRelease() (err error) {
	args, err := c.normReleaseArgs(c.config.Args)
	if err != nil {
		return
	}

	productName := args[0]
	cycle := args[1]

	response, err := c.ProductRelease(productName, cycle)
	if err != nil {
		return fmt.Errorf("failed to get release %s/%s: %w", productName, cycle, err)
	}

	c.response = response
	c.responseHeader = "Release information:"

	return
}

// HandleLatest handles the latest command.
func (c *Client) HandleLatest() (err error) {
	args := c.config.Args
	if len(args) == 0 {
		return errProductNameRequired
	}

	productName := args[0]

	response, err := c.ProductLatestRelease(productName)
	if err != nil {
		return fmt.Errorf("failed to get latest release for %s: %w", productName, err)
	}

	c.response = response
	c.responseHeader = "Latest release information:"

	return
}

// HandleCategories handles the categories command.
func (c *Client) HandleCategories() (err error) {
	args := c.config.Args
	if len(args) == 0 {
		var response *UriListResponse

		if response, err = c.Categories(); err != nil {
			return fmt.Errorf("failed to get categories: %w", err)
		}

		c.response = &CategoriesResponse{UriListResponse: response}
	} else {
		var response *ProductListResponse

		if response, err = c.ProductsByCategory(args[0]); err != nil {
			return fmt.Errorf("failed to get products for category %s: %w", args[0], err)
		}

		c.response = &CategoryProductsResponse{
			ProductListResponse: response,
			Category:            args[0],
		}
		c.responseHeader = fmt.Sprintf("Products in category '%s':", args[0])
	}

	return
}

// HandleTags handles the tags command.
func (c *Client) HandleTags() (err error) {
	args := c.config.Args
	if len(args) == 0 {
		var response *UriListResponse

		if response, err = c.Tags(); err != nil {
			return fmt.Errorf("failed to get tags: %w", err)
		}

		c.response = &TagsResponse{UriListResponse: response}
	} else {
		var response *ProductListResponse

		if response, err = c.ProductsByTag(args[0]); err != nil {
			return fmt.Errorf("failed to get products for tag %s: %w", args[0], err)
		}

		c.response = &TagProductsResponse{
			ProductListResponse: response,
			Tag:                 args[0],
		}
		c.responseHeader = fmt.Sprintf("Products with tag '%s':", args[0])
	}

	return
}

// HandleIdentifiers handles the identifiers command.
func (c *Client) HandleIdentifiers() (err error) {
	args := c.config.Args
	if len(args) == 0 {
		var response *UriListResponse

		if response, err = c.IdentifierTypes(); err != nil {
			return fmt.Errorf("failed to get identifier types: %w", err)
		}

		c.response = &IdentifierTypesResponse{UriListResponse: response}
	} else {
		var response *IdentifierListResponse

		identifierType := args[0]
		if response, err = c.IdentifiersByType(identifierType); err != nil {
			return fmt.Errorf("failed to get identifiers for type %s: %w", identifierType, err)
		}

		c.response = &TypeIdentifiersResponse{
			IdentifierListResponse: response,
			Type:                   identifierType,
		}
		c.responseHeader = fmt.Sprintf("Identifiers of type '%s':", identifierType)
	}

	return
}

// HandleCacheStats handles the cache stats command.
func (c *Client) HandleCacheStats() (err error) {
	var stats CacheStats

	if stats, err = c.cacheManager.GetStats(); err != nil {
		return fmt.Errorf("failed to get cache stats: %w", err)
	}

	c.response = &stats

	return
}

// HandleCacheClear handles the cache clear command.
func (c *Client) HandleCacheClear() (err error) {
	if err = c.cacheManager.Clear(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// Special case: cache clear just prints a message, no template formatting.
	c.Println("Cache cleared successfully")

	return
}

// HandleTemplates handles the templates list command.
func (c *Client) HandleTemplates() (err error) { //nolint:unparam // ok
	templates := c.templateManager.ListTemplates()

	c.response = &TemplateListResponse{
		Templates: templates,
		Total:     len(templates),
	}
	c.responseHeader = fmt.Sprintf("Available templates - %d total:", len(templates))

	return
}

// HandleTemplateExport handles the template export command.
func (c *Client) HandleTemplateExport() (err error) {
	args := c.config.Args[1:]
	if len(args) == 0 {
		return errOutputDirectoryRequired
	}

	outputDir := args[0]
	if err = c.templateManager.ExportTemplates(outputDir); err != nil {
		return fmt.Errorf("failed to export templates: %w", err)
	}

	c.response = &TemplateExportResponse{
		OutputDir: outputDir,
		Message:   "You can now customize the templates and use them with --template-dir",
	}
	c.responseHeader = "Templates exported to: " + outputDir

	return
}

// HandleCompletionAuto handles auto-detected shell completion.
func (c *Client) HandleCompletionAuto() (err error) { //nolint:unparam // ok
	shell := c.detectShell()
	script := c.generateCompletionScript(shell)
	c.response = &CompletionResponse{Shell: shell, Script: script}
	c.responseHeader = fmt.Sprintf("# %s completion script", shell)

	return
}

// HandleCompletionBash handles bash completion.
func (c *Client) HandleCompletionBash() (err error) { //nolint:unparam // ok
	script := c.generateCompletionScript("bash")
	c.response = &CompletionResponse{Shell: "bash", Script: script}
	c.responseHeader = "# bash completion script"

	return
}

// HandleCompletionZsh handles zsh completion.
func (c *Client) HandleCompletionZsh() (err error) { //nolint:unparam // ok
	script := c.generateCompletionScript("zsh")
	c.response = &CompletionResponse{Shell: "zsh", Script: script}
	c.responseHeader = "# zsh completion script"

	return
}

// outputJSON outputs the given data as JSON.
func (c *Client) outputJSON(data any) error {
	encoder := json.NewEncoder(c.sink)
	encoder.SetIndent("", "  ")

	return encoder.Encode(data)
}

// executeInlineTemplate executes an inline template on the given data.
func (c *Client) executeInlineTemplate(response any) (err error) {
	data := c.extractTemplateData(response)

	result, err := c.templateManager.ExecuteInline(c.config.InlineTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to execute inline template: %w", err)
	}

	c.Print(string(result))

	return
}

// extractTemplateData extracts the appropriate data from response objects for template execution.
// This function contains the shared logic used by both Format() and executeInlineTemplate().
//
//nolint:gocyclo,cyclop // ok
func (c *Client) extractTemplateData(response any) any {
	switch resp := response.(type) {
	case *IndexResponse:
		return resp.UriListResponse
	case *CategoriesResponse:
		return resp.UriListResponse
	case *TagsResponse:
		return resp.UriListResponse
	case *IdentifierTypesResponse:
		return resp.UriListResponse
	case *ProductListResponse:
		return resp
	case *FullProductListResponse:
		return resp
	case *ProductResponse:
		return &resp.Result
	case *ProductReleaseResponse:
		return &resp.Result
	case *CategoryProductsResponse:
		return struct {
			*ProductListResponse
			Category string
		}{ProductListResponse: resp.ProductListResponse, Category: resp.Category}
	case *TagProductsResponse:
		return struct {
			*ProductListResponse
			Tag string
		}{ProductListResponse: resp.ProductListResponse, Tag: resp.Tag}
	case *TypeIdentifiersResponse:
		return struct {
			*IdentifierListResponse
			Type string
		}{IdentifierListResponse: resp.IdentifierListResponse, Type: resp.Type}
	case *CacheStats:
		return resp
	case *TemplateListResponse:
		return resp
	case *TemplateExportResponse:
		return resp
	case *CompletionResponse:
		return resp
	default:
		return response
	}
}

func (c *Client) normReleaseArgs(args []string) (ret []string, err error) {
	if len(args) < 2 {
		return nil, errProductReleaseNameRequired
	}

	originalVersion := args[1]
	normalizedVersion := extractMajorMinor(originalVersion)

	//nolint:godox,staticcheck // ok
	if originalVersion != normalizedVersion && isSemanticVersion(originalVersion) && c.config.IsText() {
		// TODO: Re-enable with verbose/quiet flags - breaks clean output for templates and JSON piping
		//    c.Printf("Note: Normalized version %s to %s for API compatibility\n\n", originalVersion, normalizedVersion)
	}

	ret = slices.Clone(args)
	ret[1] = normalizedVersion

	return
}

// detectShell auto-detects the current shell from environment.
func (c *Client) detectShell() (name string) {
	shell := cmp.Or(os.Getenv("SHELL"), "bash")
	switch name = path.Base(shell); name {
	case "bash", "zsh":
		return name
	default:
		return "bash"
	}
}

// generateCompletionScript generates shell completion script.
func (c *Client) generateCompletionScript(shell string) string {
	switch shell {
	case "zsh":
		return zshCompletionScript
	default:
		return bashCompletionScript
	}
}

// preRouting flattens subcommands into path-based routing.
func (c *Client) preRouting(cmd string) string {
	switch args := c.config.Args; cmd {
	case "cache":
		if len(args) > 0 {
			return "cache/" + args[0]
		}

		return "cache/"
	case "templates":
		if len(args) > 0 && args[0] == "export" {
			return "templates/export"
		}

		return "templates/list"
	case "completion":
		if len(args) > 0 {
			return "completion/" + args[0]
		}

		return "completion/"
	default:
		return cmd
	}
}
