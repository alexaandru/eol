package eol

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	errProductNameEmpty    = errors.New("product name cannot be empty")
	errReleaseNameEmpty    = errors.New("release name cannot be empty")
	errCategoryNameEmpty   = errors.New("category name cannot be empty")
	errTagNameEmpty        = errors.New("tag name cannot be empty")
	errIdentifierTypeEmpty = errors.New("identifier type cannot be empty")
)

// Index returns the main endoflife.date API endpoints.
func (c *Client) Index() (r *UriListResponse, err error) {
	r = &UriListResponse{}
	if err = c.doRequest("/", r); err != nil {
		return nil, fmt.Errorf("failed to get API index: %w", err)
	}

	return
}

// Products returns a list of all available products.
func (c *Client) Products() (r *ProductListResponse, err error) {
	r = &ProductListResponse{}

	// Smart caching: First check if we can get the products list from cached ProductsFull data.
	if cachedProducts, found := c.cacheManager.GetProductsFromFullCache(); found {
		if err = json.Unmarshal(cachedProducts, r); err == nil {
			return r, nil // Cache hit from full products data.
		}
	}

	// If not found in full cache, proceed with normal API call.
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
func (c *Client) Product(product string) (r *ProductResponse, err error) {
	if product == "" {
		return nil, errProductNameEmpty
	}

	r = &ProductResponse{}

	// Smart caching: First check if we can get the product from cached ProductsFull data.
	if cachedProduct, found := c.cacheManager.GetProductFromFullCache(product); found {
		if err = json.Unmarshal(cachedProduct, r); err == nil {
			return r, nil // Cache hit from full products data.
		}
	}

	// If not found in full cache, proceed with normal API call.
	endpoint := "/products/" + product
	if err = c.doRequest(endpoint, r, product); err != nil {
		return nil, fmt.Errorf("failed to get product %s: %w", product, err)
	}

	return
}

// ProductRelease returns information about a specific product release cycle.
func (c *Client) ProductRelease(product, release string) (r *ProductReleaseResponse, err error) { //nolint:varnamelen,lll // ok
	if product == "" {
		return nil, errProductNameEmpty
	}

	if release == "" {
		return nil, errReleaseNameEmpty
	}

	r = &ProductReleaseResponse{}

	// Smart caching: First check if we can get the release from cached ProductsFull data.
	if cachedRelease, found := c.cacheManager.GetReleaseFromFullCache(product, release); found {
		if err = json.Unmarshal(cachedRelease, r); err == nil {
			return r, nil // Cache hit from full products data.
		}
	}

	// Fallback: Check if we can get the release from cached individual product data.
	if cachedRelease, found := c.cacheManager.GetReleaseFromProductCache(product, release); found {
		if err = json.Unmarshal(cachedRelease, r); err == nil {
			return r, nil // Cache hit from product data.
		}
	}

	// If not found in product cache, proceed with normal API calls.
	var (
		normalizedRelease = normalizeVersion(release)
		endpoint          = fmt.Sprintf("/products/%s/releases/%s", product, normalizedRelease)
	)

	err = c.doRequest(endpoint, r, product, normalizedRelease)
	if err != nil {
		// If normalization was applied and it failed, try with original version.
		if normalizedRelease != release {
			originalEndpoint := fmt.Sprintf("/products/%s/releases/%s", product, release)
			if originalErr := c.doRequest(originalEndpoint, &r, product, release); originalErr == nil {
				return r, nil
			}

			// Both attempts failed, provide helpful error message.
			return nil, fmt.Errorf("failed to get release %s for product %s (also tried %s): %w",
				release, product, normalizedRelease, err)
		}

		return nil, fmt.Errorf("failed to get release %s for product %s: %w", release, product, err)
	}

	return
}

// ProductLatestRelease returns information about the latest release cycle for a product.
func (c *Client) ProductLatestRelease(product string) (r *ProductReleaseResponse, err error) {
	if product == "" {
		return nil, errProductNameEmpty
	}

	r = &ProductReleaseResponse{}

	endpoint := fmt.Sprintf("/products/%s/releases/latest", product)
	if err = c.doRequest(endpoint, r, product, "latest"); err != nil {
		return nil, fmt.Errorf("failed to get latest release for product %s: %w", product, err)
	}

	return
}

// Categories returns a list of all categories.
func (c *Client) Categories() (r *UriListResponse, err error) {
	r = &UriListResponse{}
	if err = c.doRequest("/categories", r); err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return
}

// ProductsByCategory returns all products in a specific category.
func (c *Client) ProductsByCategory(category string) (r *ProductListResponse, err error) {
	if category == "" {
		return nil, errCategoryNameEmpty
	}

	r = &ProductListResponse{}

	endpoint := "/categories/" + category
	if err = c.doRequest(endpoint, r, "category", category); err != nil {
		return nil, fmt.Errorf("failed to get products for category %s: %w", category, err)
	}

	return
}

// Tags returns a list of all tags.
func (c *Client) Tags() (r *UriListResponse, err error) {
	r = &UriListResponse{}
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

	endpoint := "/tags/" + tag
	if err = c.doRequest(endpoint, r, "tag", tag); err != nil {
		return nil, fmt.Errorf("failed to get products for tag %s: %w", tag, err)
	}

	return
}

// IdentifierTypes returns a list of all identifier types.
func (c *Client) IdentifierTypes() (r *UriListResponse, err error) {
	r = &UriListResponse{}
	if err = c.doRequest("/identifiers", r); err != nil {
		return nil, fmt.Errorf("failed to get identifier types: %w", err)
	}

	return
}

// IdentifiersByType returns all identifiers for a given type.
func (c *Client) IdentifiersByType(identifierType string) (r *IdentifierListResponse, err error) {
	if identifierType == "" {
		return nil, errIdentifierTypeEmpty
	}

	r = &IdentifierListResponse{}

	endpoint := "/identifiers/" + identifierType
	if err = c.doRequest(endpoint, r, "identifier", identifierType); err != nil {
		return nil, fmt.Errorf("failed to get identifiers for type %s: %w", identifierType, err)
	}

	return
}
