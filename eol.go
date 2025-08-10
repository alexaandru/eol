package eol

import (
	"time"
)

// Uri represents a link to a resource.
type Uri struct {
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

// UriListResponse represents a response containing a list of URIs.
type UriListResponse struct {
	SchemaVersion string `json:"schema_version"`
	Result        []Uri  `json:"result"`
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
	Product    Uri    `json:"product"`
}

// IdentifierListResponse represents a response containing identifiers for a given type.
type IdentifierListResponse struct {
	SchemaVersion string              `json:"schema_version"`
	Result        []IdentifierProduct `json:"result"`
	Total         int                 `json:"total"`
}
