package eol

import (
	"errors"
	"fmt"
)

var (
	errNotFound            = errors.New("not found")
	errProductNameEmpty    = errors.New("product name cannot be empty")
	errReleaseNameEmpty    = errors.New("release name cannot be empty")
	errCategoryNameEmpty   = errors.New("category name cannot be empty")
	errTagNameEmpty        = errors.New("tag name cannot be empty")
	errIdentifierTypeEmpty = errors.New("identifier type cannot be empty")
)

// Index returns the main endoflife.date API endpoints.
func (c *Client) Index() (r *URIListResponse, err error) {
	r = &URIListResponse{}
	if err = c.doRequest("/", r); err != nil {
		return nil, fmt.Errorf("failed to get API index: %w", err)
	}

	return
}

// Products returns a list of all available products.
func (c *Client) Products() (r *ProductListResponse, err error) {
	r = &ProductListResponse{}
	if err = c.doRequest("/products", r); err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	return
}

// ProductsFull returns a list of all products with full details.
func (c *Client) ProductsFull() (r *FullProductListResponse, err error) {
	r = &FullProductListResponse{}
	if err = c.doRequest("/products/full", r); err != nil {
		return nil, fmt.Errorf("failed to get full products: %w", err)
	}

	return
}

// Product returns details for a specific product.
func (c *Client) Product(p string) (r *ProductResponse, err error) {
	if p == "" {
		return nil, errProductNameEmpty
	}

	r = &ProductResponse{}
	if err = c.doRequest("/products/"+p, r, p); err != nil {
		return nil, fmt.Errorf("failed to get product %s: %w", p, err)
	}

	return
}

// ProductRelease returns information about a specific product release cycle.
func (c *Client) ProductRelease(p, rls string) (r *ProductReleaseResponse, err error) {
	if p == "" {
		return nil, errProductNameEmpty
	}

	if rls == "" {
		return nil, errReleaseNameEmpty
	}

	r = &ProductReleaseResponse{}

	variants := generateVersionVariants(rls)
	for _, variant := range variants {
		err = c.doRequest("/products/"+p+"/releases/"+variant, r, p, variant)
		if err == nil {
			return //nolint:nilerr // ok
		}

		if !errors.Is(err, errNotFound) {
			return nil, fmt.Errorf("failed to get release %s for product %s: %w", rls, p, err)
		}
	}

	return nil, fmt.Errorf("failed to get release %s for product %s (variants: %v): %w",
		rls, p, variants, err)
}

// ProductLatestRelease returns information about the latest release cycle for a product.
func (c *Client) ProductLatestRelease(p string) (r *ProductReleaseResponse, err error) {
	if p == "" {
		return nil, errProductNameEmpty
	}

	r = &ProductReleaseResponse{}
	if err = c.doRequest("/products/"+p+"/releases/latest", r, p, "latest"); err != nil {
		return nil, fmt.Errorf("failed to get latest release for product %s: %w", p, err)
	}

	return
}

// Categories returns a list of all categories.
func (c *Client) Categories() (r *URIListResponse, err error) {
	r = &URIListResponse{}
	if err = c.doRequest("/categories", r); err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return
}

// ProductsByCategory returns all products in a specific category.
func (c *Client) ProductsByCategory(cat string) (r *ProductListResponse, err error) {
	if cat == "" {
		return nil, errCategoryNameEmpty
	}

	r = &ProductListResponse{}
	if err = c.doRequest("/categories/"+cat, r, "category", cat); err != nil {
		return nil, fmt.Errorf("failed to get products for category %s: %w", cat, err)
	}

	return
}

// Tags returns a list of all tags.
func (c *Client) Tags() (r *URIListResponse, err error) {
	r = &URIListResponse{}
	if err = c.doRequest("/tags", r); err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	return
}

// ProductsByTag returns all products with a specific tag.
func (c *Client) ProductsByTag(tag string) (r *ProductListResponse, err error) {
	if tag == "" {
		return nil, errTagNameEmpty
	}

	r = &ProductListResponse{}
	if err = c.doRequest("/tags/"+tag, r, "tag", tag); err != nil {
		return nil, fmt.Errorf("failed to get products for tag %s: %w", tag, err)
	}

	return
}

// IdentifierTypes returns a list of all identifier types.
func (c *Client) IdentifierTypes() (r *URIListResponse, err error) {
	r = &URIListResponse{}
	if err = c.doRequest("/identifiers", r); err != nil {
		return nil, fmt.Errorf("failed to get identifier types: %w", err)
	}

	return
}

// IdentifiersByType returns all identifiers for a given type.
func (c *Client) IdentifiersByType(typ string) (r *IdentifierListResponse, err error) {
	if typ == "" {
		return nil, errIdentifierTypeEmpty
	}

	r = &IdentifierListResponse{}
	if err = c.doRequest("/identifiers/"+typ, r, "identifier", typ); err != nil {
		return nil, fmt.Errorf("failed to get identifiers for type %s: %w", typ, err)
	}

	return
}
