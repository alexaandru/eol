package eol

import (
	"time"
)

// URI represents a link to a resource.
type URI struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// Identifier represents a product identifier (purl, cpe, etc.)
type Identifier struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// ProductVersion contains information about a specific product version.
type ProductVersion struct {
	Date *string `json:"date"`
	Link *string `json:"link"`
	Name string  `json:"name"`
}

// ProductRelease contains full information about a product release cycle.
type ProductRelease struct {
	EolFrom          *string         `json:"eolFrom"`
	IsDiscontinued   *bool           `json:"isDiscontinued,omitempty"`
	EoesFrom         *string         `json:"eoesFrom,omitempty"`
	EoasFrom         *string         `json:"eoasFrom,omitempty"`
	DiscontinuedFrom *string         `json:"discontinuedFrom,omitempty"`
	LtsFrom          *string         `json:"ltsFrom"`
	Custom           map[string]any  `json:"custom,omitempty"`
	IsEoes           *bool           `json:"isEoes,omitempty"`
	Codename         *string         `json:"codename"`
	Latest           *ProductVersion `json:"latest"`
	IsEoas           *bool           `json:"isEoas,omitempty"`
	Name             string          `json:"name"`
	ReleaseDate      string          `json:"releaseDate"`
	Label            string          `json:"label"`
	IsLts            bool            `json:"isLts"`
	IsMaintained     bool            `json:"isMaintained"`
	IsEol            bool            `json:"isEol"`
}

// CliProductRelease is a CLI-friendly version of ProductRelease with regular bool fields.
//
//nolint:govet // ok
type CliProductRelease struct {
	Name             string          `json:"name"`
	Label            string          `json:"label"`
	ReleaseDate      string          `json:"releaseDate"`
	IsLts            bool            `json:"isLts"`
	IsEol            bool            `json:"isEol"`
	IsMaintained     bool            `json:"isMaintained"`
	IsEoas           bool            `json:"isEoas"`
	IsDiscontinued   bool            `json:"isDiscontinued"`
	IsEoes           bool            `json:"isEoes"`
	EolFrom          string          `json:"eolFrom"`
	LtsFrom          string          `json:"ltsFrom"`
	EoasFrom         string          `json:"eoasFrom"`
	EoesFrom         string          `json:"eoesFrom"`
	DiscontinuedFrom string          `json:"discontinuedFrom"`
	Codename         string          `json:"codename"`
	Latest           *ProductVersion `json:"latest"`
	Custom           map[string]any  `json:"custom,omitempty"`
}

// ProductSummary contains basic information about a product.
type ProductSummary struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Category string   `json:"category"`
	URI      string   `json:"uri"`
	Aliases  []string `json:"aliases"`
	Tags     []string `json:"tags"`
}

// ProductLabels contains the labels used for different phases.
type ProductLabels struct {
	Eoas         *string `json:"eoas"`
	Discontinued *string `json:"discontinued"`
	Eoes         *string `json:"eoes"`
	Eol          string  `json:"eol"`
}

// ProductLinks contains various links related to the product.
type ProductLinks struct {
	Icon          *string `json:"icon"`
	ReleasePolicy *string `json:"releasePolicy"`
	HTML          string  `json:"html"`
}

// ProductDetails contains full details about a product.
type ProductDetails struct {
	Name           string           `json:"name"`
	Label          string           `json:"label"`
	Aliases        []string         `json:"aliases"`
	Category       string           `json:"category"`
	Tags           []string         `json:"tags"`
	VersionCommand *string          `json:"versionCommand"`
	Identifiers    []Identifier     `json:"identifiers"`
	Labels         ProductLabels    `json:"labels"`
	Links          ProductLinks     `json:"links"`
	Releases       []ProductRelease `json:"releases"`
}

// URIListResponse represents a response containing a list of URIs.
type URIListResponse struct {
	SchemaVersion string `json:"schema_version"`
	Result        []URI  `json:"result"`
	Total         int    `json:"total"`
}

// ProductListResponse represents a response containing a list of product summaries.
type ProductListResponse struct {
	SchemaVersion string           `json:"schema_version"`
	Result        []ProductSummary `json:"result"`
	Total         int              `json:"total"`
}

// FullProductListResponse represents a response containing a list of full product details.
type FullProductListResponse struct {
	SchemaVersion string           `json:"schema_version"`
	Result        []ProductDetails `json:"result"`
	Total         int              `json:"total"`
}

// ProductResponse represents a response containing a single product.
type ProductResponse struct {
	SchemaVersion string         `json:"schema_version"`
	LastModified  time.Time      `json:"last_modified"`
	Result        ProductDetails `json:"result"`
}

// ProductReleaseResponse represents a response containing a single release cycle.
type ProductReleaseResponse struct {
	SchemaVersion string         `json:"schema_version"`
	Result        ProductRelease `json:"result"`
}

// IdentifierProduct represents the product reference in an identifier response.
type IdentifierProduct struct {
	Identifier string `json:"identifier"`
	Product    URI    `json:"product"`
}

// IdentifierListResponse represents a response containing identifiers for a given type.
type IdentifierListResponse struct {
	SchemaVersion string              `json:"schema_version"`
	Result        []IdentifierProduct `json:"result"`
	Total         int                 `json:"total"`
}

// ToCliRelease converts a ProductRelease to a CliProductRelease with clean bool fields.
func (pr *ProductRelease) ToCliRelease() (cli CliProductRelease) {
	cli = CliProductRelease{
		Name:         pr.Name,
		Label:        pr.Label,
		ReleaseDate:  pr.ReleaseDate,
		IsLts:        pr.IsLts,
		IsEol:        pr.IsEol,
		IsMaintained: pr.IsMaintained,
		Latest:       pr.Latest,
		Custom:       pr.Custom,
	}

	if pr.EolFrom != nil {
		cli.EolFrom = *pr.EolFrom
	}

	if pr.LtsFrom != nil {
		cli.LtsFrom = *pr.LtsFrom
	}

	if pr.EoasFrom != nil {
		cli.EoasFrom = *pr.EoasFrom
	}

	if pr.EoesFrom != nil {
		cli.EoesFrom = *pr.EoesFrom
	}

	if pr.DiscontinuedFrom != nil {
		cli.DiscontinuedFrom = *pr.DiscontinuedFrom
	}

	if pr.Codename != nil {
		cli.Codename = *pr.Codename
	}

	if pr.IsEoas != nil {
		cli.IsEoas = *pr.IsEoas
	}

	if pr.IsDiscontinued != nil {
		cli.IsDiscontinued = *pr.IsDiscontinued
	}

	if pr.IsEoes != nil {
		cli.IsEoes = *pr.IsEoes
	}

	return
}
