package eol

import (
	"cmp"
	"crypto/md5" //nolint:gosec // fine for cache keys
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

// CacheManager handles caching of API responses.
type CacheManager struct {
	baseDir    string
	baseURL    string
	enabled    bool
	defaultTTL time.Duration
	fullTTL    time.Duration
}

// CacheStrategy represents a cache lookup strategy with extraction logic.
type CacheStrategy struct { //nolint:govet // ok
	CacheKey    string
	ExtractFunc func(json.RawMessage, ...string) (json.RawMessage, bool)
}

// CacheEntry represents a cached API response.
type CacheEntry struct {
	Timestamp  time.Time       `json:"timestamp"`
	ExpiresAt  time.Time       `json:"expires_at"`
	Endpoint   string          `json:"endpoint"`
	Parameters string          `json:"parameters"`
	Data       json.RawMessage `json:"data"`
}

// CacheStats represents cache statistics.
type CacheStats struct {
	CacheDir     string `json:"cache_dir"`
	DefaultTTL   string `json:"default_ttl"`
	FullTTL      string `json:"full_ttl"`
	TotalSize    int    `json:"total_size"`
	TotalFiles   int    `json:"total_files"`
	ExpiredFiles int    `json:"expired_files"`
	ValidFiles   int    `json:"valid_files"`
	Enabled      bool   `json:"enabled"`
}

const (
	fullTTL  = 24 * time.Hour // The TTL used for full endpoints (e.g., /products/full).
	cacheExt = ".eol_cache.json"
)

var errRefusingToClear = errors.New("refusing to clear")

// NewCacheManager creates a new cache manager.
func NewCacheManager(baseDir, baseURL string, enabled bool, defaultTTL time.Duration) *CacheManager {
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		switch {
		case err != nil:
			baseDir = ".eol-cache"
		case runtime.GOOS == "windows":
			baseDir = filepath.Join(homeDir, "AppData", "Local", "eol-cache")
		case runtime.GOOS == "darwin":
			baseDir = filepath.Join(homeDir, "Library", "Caches", "eol")
		default:
			baseDir = filepath.Join(homeDir, ".cache", "eol")
		}
	}

	return &CacheManager{
		baseDir:    baseDir,
		baseURL:    baseURL,
		enabled:    enabled,
		defaultTTL: defaultTTL,
		fullTTL:    fullTTL,
	}
}

// Get retrieves data from cache using smart strategy hierarchy.
func (cm *CacheManager) Get(endpoint string, params ...string) (_ json.RawMessage, found bool) {
	// For --full endpoints, always check cache regardless of enabled flag.
	if !cm.enabled && !cm.isFullEndpoint(endpoint) {
		return
	}

	strategies := cm.buildCacheStrategies(endpoint, params...)

	return cm.tryStrategies(strategies, 0, params...)
}

// Set stores data in cache.
func (cm *CacheManager) Set(endpoint string, data any, params ...string) (err error) {
	// For --full endpoints, always cache regardless of enabled flag.
	if !cm.enabled && !cm.isFullEndpoint(endpoint) {
		return
	}

	if err = cm.ensureCacheDir(); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	ttl := cm.defaultTTL
	if cm.isFullEndpoint(endpoint) {
		ttl = cm.fullTTL
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for caching: %w", err)
	}

	now := time.Now()
	entry := CacheEntry{
		Timestamp:  now,
		ExpiresAt:  now.Add(ttl),
		Data:       json.RawMessage(rawData),
		Endpoint:   endpoint,
		Parameters: strings.Join(params, "|"),
	}

	jsonData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	key := cm.generateCacheKey(endpoint, params...)
	filePath := cm.getCacheFilePath(key)

	if err = os.WriteFile(filePath, jsonData, filePerm); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return
}

// Clear removes all cache files, safely.
func (cm *CacheManager) Clear() (err error) {
	allowedDirs := []string{".eol-cache", "eol-cache", "eol"}
	if dirName := filepath.Base(cm.baseDir); !slices.Contains(allowedDirs, dirName) {
		return fmt.Errorf("%w non-default cache folder: %q", errRefusingToClear, dirName)
	}

	matches, err := filepath.Glob(filepath.Join(cm.baseDir, "*"+cacheExt))
	if err != nil {
		return fmt.Errorf("failed to find cache files: %w", err)
	}

	for _, file := range matches {
		if err = os.Remove(file); err != nil {
			return fmt.Errorf("failed to remove cache file %s: %w", file, err)
		}
	}

	return
}

// ClearExpired removes expired cache files.
func (cm *CacheManager) ClearExpired() (err error) {
	if err = cm.ensureCacheDir(); err != nil {
		return
	}

	entries, err := os.ReadDir(cm.baseDir)
	if err != nil {
		return
	}

	now := time.Now()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), cacheExt) {
			continue
		}

		filePath := filepath.Join(cm.baseDir, entry.Name())

		var data []byte

		//nolint:gosec // Reading cache file is safe
		if data, err = os.ReadFile(filePath); err != nil {
			continue
		}

		cacheEntry := CacheEntry{}
		if err = json.Unmarshal(data, &cacheEntry); err != nil {
			continue
		}

		if now.After(cacheEntry.ExpiresAt) {
			os.Remove(filePath) //nolint:errcheck,gosec // TODO
		}
	}

	return
}

// GetStats returns cache statistics.
func (cm *CacheManager) GetStats() (stats CacheStats, err error) {
	if err = cm.ensureCacheDir(); err != nil {
		return
	}

	entries, err := os.ReadDir(cm.baseDir)
	if err != nil {
		return
	}

	stats = CacheStats{
		Enabled:    cm.enabled,
		CacheDir:   cm.baseDir,
		DefaultTTL: cm.defaultTTL.String(),
		FullTTL:    cm.fullTTL.String(),
	}
	now := time.Now()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(cm.baseDir, entry.Name())

		var fileInfo os.FileInfo

		if fileInfo, err = entry.Info(); err != nil {
			continue
		}

		stats.TotalFiles++
		stats.TotalSize += int(fileInfo.Size())

		var data []byte

		//nolint:gosec // Reading cache file is safe
		if data, err = os.ReadFile(filePath); err != nil {
			continue
		}

		cacheEntry := CacheEntry{}
		if err = json.Unmarshal(data, &cacheEntry); err != nil {
			continue
		}

		if now.After(cacheEntry.ExpiresAt) {
			stats.ExpiredFiles++
		} else {
			stats.ValidFiles++
		}
	}

	return
}

// MustUseCache returns true if this endpoint must use cache (like --full).
func (cm *CacheManager) MustUseCache(endpoint string) bool {
	return cm.isFullEndpoint(endpoint)
}

// getRawCacheByKey retrieves raw cache data with TTL validation using a generated cache key.
func (cm *CacheManager) getRawCacheByKey(cacheKey string) (_ json.RawMessage, found bool) {
	filePath := cm.getCacheFilePath(cacheKey)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return
	}

	data, err := os.ReadFile(filePath) //nolint:gosec // Reading cache file is safe
	if err != nil {
		return
	}

	entry := CacheEntry{}
	if err = json.Unmarshal(data, &entry); err != nil {
		return
	}

	if time.Now().After(entry.ExpiresAt) {
		os.Remove(filePath) //nolint:errcheck,gosec // TODO
		return
	}

	return entry.Data, true
}

// buildCacheStrategies creates the ordered list of cache strategies for an endpoint.
func (cm *CacheManager) buildCacheStrategies(endpoint string, params ...string) []CacheStrategy {
	switch {
	case endpoint == "/products":
		return []CacheStrategy{
			{cm.generateCacheKey("/products"), cm.extractExact},
			{cm.generateCacheKey("/products/full"), cm.extractProductsFromFull},
		}
	case strings.HasPrefix(endpoint, "/products/") && len(params) >= 1 && !strings.Contains(endpoint, "/releases/"):
		p := params[0]

		return []CacheStrategy{
			{cm.generateCacheKey("/products/"+p, p), cm.extractExact},
			{cm.generateCacheKey("/products/full"), func(data json.RawMessage, _ ...string) (json.RawMessage, bool) {
				return cm.extractProductFromFull(data, p)
			}},
		}
	case strings.HasPrefix(endpoint, "/products/") && strings.Contains(endpoint, "/releases/") && len(params) >= 2:
		p, rel := params[0], params[1]

		return []CacheStrategy{
			{cm.generateCacheKey("/products/"+p+"/releases/"+rel, p, rel), cm.extractExact},
			{
				cm.generateCacheKey("/products/"+p, p),
				func(data json.RawMessage, _ ...string) (json.RawMessage, bool) {
					return cm.extractReleaseFromProduct(data, rel)
				},
			},
			{
				cm.generateCacheKey("/products/full"),
				func(data json.RawMessage, _ ...string) (json.RawMessage, bool) {
					return cm.extractReleaseFromFull(data, p, rel)
				},
			},
		}
	case endpoint == "/categories":
		return []CacheStrategy{
			{cm.generateCacheKey("/categories"), cm.extractExact},
			{cm.generateCacheKey("/products/full"), cm.extractCategoriesFromFull},
		}
	case strings.HasPrefix(endpoint, "/categories/") && len(params) >= 2:
		cat := params[1] // If params[0] is "category", params[1] is the actual category.

		return []CacheStrategy{
			{cm.generateCacheKey("/categories/"+cat, "category", cat), cm.extractExact},
			{cm.generateCacheKey("/products/full"), func(data json.RawMessage, _ ...string) (json.RawMessage, bool) {
				return cm.extractProductsByCategoryFromFull(data, cat)
			}},
		}
	case endpoint == "/tags":
		return []CacheStrategy{
			{cm.generateCacheKey("/tags"), cm.extractExact},
			{cm.generateCacheKey("/products/full"), cm.extractTagsFromFull},
		}
	case strings.HasPrefix(endpoint, "/tags/") && len(params) >= 2:
		tag := params[1] // If params[0] is "tag", params[1] is the actual tag.

		return []CacheStrategy{
			{cm.generateCacheKey("/tags/"+tag, "tag", tag), cm.extractExact},
			{cm.generateCacheKey("/products/full"), func(data json.RawMessage, _ ...string) (json.RawMessage, bool) {
				return cm.extractProductsByTagFromFull(data, tag)
			}},
		}
	default:
		// For all other endpoints (/, /identifiers, /identifiers/{type}), only exact cache.
		return []CacheStrategy{{cm.generateCacheKey(endpoint), cm.extractExact}}
	}
}

// tryStrategies recursively tries cache strategies from smallest to largest.
//
//nolint:lll // ok
func (cm *CacheManager) tryStrategies(strategies []CacheStrategy, index int, params ...string) (_ json.RawMessage, found bool) {
	if index >= len(strategies) {
		return
	}

	strategy := strategies[index]
	if cached, ok := cm.getRawCacheByKey(strategy.CacheKey); ok {
		if data, extracted := strategy.ExtractFunc(cached, params...); extracted {
			return data, true
		}
	}

	return cm.tryStrategies(strategies, index+1, params...)
}

// extractExact returns the cached data as-is (no extraction needed).
func (cm *CacheManager) extractExact(data json.RawMessage, params ...string) (_ json.RawMessage, found bool) {
	return data, true
}

// extractProductsFromFull extracts a products list from full products cache.
//
//nolint:lll // ok
func (cm *CacheManager) extractProductsFromFull(data json.RawMessage, params ...string) (_ json.RawMessage, found bool) {
	fullResponse := map[string]any{}
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return
	}

	products := make([]map[string]any, 0, len(result))

	for _, p := range result {
		productMap, ok2 := p.(map[string]any)
		if !ok2 {
			continue
		}

		summary := map[string]any{
			"name":     productMap["name"],
			"label":    productMap["label"],
			"category": productMap["category"],
			"uri":      fmt.Sprintf("%s/products/%s", cm.baseURL, productMap["name"]),
		}

		if aliases, ok3 := productMap["aliases"]; ok3 {
			summary["aliases"] = aliases
		}

		if tags, ok4 := productMap["tags"]; ok4 {
			summary["tags"] = tags
		}

		products = append(products, summary)
	}

	productsResponse := map[string]any{
		"schema_version": fullResponse["schema_version"],
		"total":          len(products),
		"result":         products,
	}

	productsJSON, err := json.Marshal(productsResponse)
	if err != nil {
		return
	}

	return productsJSON, true
}

// extractProductFromFull extracts a specific product from full products cache.
func (cm *CacheManager) extractProductFromFull(data json.RawMessage, product string) (_ json.RawMessage, found bool) {
	fullResponse := map[string]any{}
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return
	}

	for _, p := range result {
		productMap, ok2 := p.(map[string]any)
		if !ok2 {
			continue
		}

		name, ok2 := productMap["name"].(string)
		if !ok2 || name != product {
			continue
		}

		productResponse := map[string]any{
			"schema_version": fullResponse["schema_version"],
			"last_modified":  "2025-01-11T00:00:00Z",
			"result":         productMap,
		}

		productJSON, err := json.Marshal(productResponse)
		if err != nil {
			return
		}

		return productJSON, true
	}

	return
}

// extractReleaseFromProduct extracts a specific release from product cache.
//
//nolint:lll // ok
func (cm *CacheManager) extractReleaseFromProduct(data json.RawMessage, release string) (_ json.RawMessage, found bool) {
	fullProductResponse := map[string]any{}
	if err := json.Unmarshal(data, &fullProductResponse); err != nil {
		return
	}

	result, ok := fullProductResponse["result"].(map[string]any)
	if !ok {
		return
	}

	releases, ok := result["releases"].([]any)
	if !ok {
		return
	}

	for _, r := range releases {
		releaseMap, ok2 := r.(map[string]any)
		if !ok2 {
			continue
		}

		name, ok2 := releaseMap["name"].(string)
		if !ok2 {
			continue
		}

		if name == release || name == normalizeVersion(release) {
			releaseResponse := map[string]any{
				"schema_version": fullProductResponse["schema_version"],
				"result":         releaseMap,
			}

			releaseJSON, err := json.Marshal(releaseResponse)
			if err != nil {
				return
			}

			return releaseJSON, true
		}
	}

	return
}

// extractReleaseFromFull extracts a specific release from full products cache.
//
//nolint:gocognit,lll // ok
func (cm *CacheManager) extractReleaseFromFull(data json.RawMessage, product, release string) (_ json.RawMessage, found bool) {
	fullResponse := map[string]any{}
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return
	}

	for _, p := range result {
		productMap, ok2 := p.(map[string]any)
		if !ok2 {
			continue
		}

		name, ok2 := productMap["name"].(string)
		if !ok2 || name != product {
			continue
		}

		releases, ok2 := productMap["releases"].([]any)
		if !ok2 {
			continue
		}

		for _, r := range releases {
			releaseMap, ok3 := r.(map[string]any)
			if !ok3 {
				continue
			}

			releaseName, ok3 := releaseMap["name"].(string)
			if !ok3 {
				continue
			}

			if releaseName == release || releaseName == normalizeVersion(release) {
				releaseResponse := map[string]any{
					"schema_version": fullResponse["schema_version"],
					"result":         releaseMap,
				}

				releaseJSON, err := json.Marshal(releaseResponse)
				if err != nil {
					return
				}

				return releaseJSON, true
			}
		}

		return
	}

	return
}

// extractCategoriesFromFull extracts unique categories from full products cache.
//
//nolint:lll // ok
func (cm *CacheManager) extractCategoriesFromFull(data json.RawMessage, params ...string) (_ json.RawMessage, found bool) {
	fullResponse := map[string]any{}
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return
	}

	categorySet := map[string]bool{}
	for _, p := range result {
		productMap, ok2 := p.(map[string]any)
		if !ok2 {
			continue
		}

		if category, ok3 := productMap["category"].(string); ok3 && category != "" {
			categorySet[category] = true
		}
	}

	categories := make([]map[string]any, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, map[string]any{
			"name": cat,
			"uri":  cm.baseURL + "/categories/" + cat,
		})
	}

	categoriesResponse := map[string]any{
		"schema_version": fullResponse["schema_version"],
		"total":          len(categories),
		"result":         categories,
	}

	categoriesJSON, err := json.Marshal(categoriesResponse)
	if err != nil {
		return
	}

	return categoriesJSON, true
}

// extractProductsByCategoryFromFull extracts products by category from full products cache.
//
//nolint:lll // ok
func (cm *CacheManager) extractProductsByCategoryFromFull(data json.RawMessage, category string) (_ json.RawMessage, found bool) {
	fullResponse := map[string]any{}
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return
	}

	products := []map[string]any{}
	for _, p := range result {
		productMap, ok2 := p.(map[string]any)
		if !ok2 {
			continue
		}

		if productCategory, ok3 := productMap["category"].(string); ok3 && productCategory == category {
			summary := map[string]any{
				"name":     productMap["name"],
				"label":    productMap["label"],
				"category": productMap["category"],
				"uri":      fmt.Sprintf("%s/products/%s", cm.baseURL, productMap["name"]),
			}

			if aliases, ok4 := productMap["aliases"]; ok4 {
				summary["aliases"] = aliases
			}

			if tags, ok4 := productMap["tags"]; ok4 {
				summary["tags"] = tags
			}

			products = append(products, summary)
		}
	}

	productsResponse := map[string]any{
		"schema_version": fullResponse["schema_version"],
		"total":          len(products),
		"result":         products,
	}

	productsJSON, err := json.Marshal(productsResponse)
	if err != nil {
		return
	}

	return productsJSON, true
}

// extractTagsFromFull extracts unique tags from full products cache.
//
//nolint:gocognit // ok
func (cm *CacheManager) extractTagsFromFull(data json.RawMessage, params ...string) (_ json.RawMessage, found bool) {
	fullResponse := map[string]any{}
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return
	}

	tagSet := map[string]bool{}
	for _, p := range result {
		productMap, ok2 := p.(map[string]any)
		if !ok2 {
			continue
		}

		if tagsInterface, ok3 := productMap["tags"]; ok3 {
			if tags, ok4 := tagsInterface.([]any); ok4 {
				for _, tagInterface := range tags {
					if tag, ok5 := tagInterface.(string); ok5 && tag != "" {
						tagSet[tag] = true
					}
				}
			}
		}
	}

	tags := make([]map[string]any, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, map[string]any{
			"name": tag,
			"uri":  cm.baseURL + "/tags/" + tag,
		})
	}

	tagsResponse := map[string]any{
		"schema_version": fullResponse["schema_version"],
		"total":          len(tags),
		"result":         tags,
	}

	tagsJSON, err := json.Marshal(tagsResponse)
	if err != nil {
		return
	}

	return tagsJSON, true
}

// extractProductsByTagFromFull extracts products by tag from full products cache.
//
//nolint:gocognit // ok
func (cm *CacheManager) extractProductsByTagFromFull(data json.RawMessage, tag string) (_ json.RawMessage, found bool) {
	fullResponse := map[string]any{}
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return
	}

	products := []map[string]any{}
	for _, p := range result {
		productMap, ok2 := p.(map[string]any)
		if !ok2 {
			continue
		}

		hasTag := false

		if tagsInterface, ok3 := productMap["tags"]; ok3 {
			if tags, ok4 := tagsInterface.([]any); ok4 {
				for _, tagInterface := range tags {
					if productTag, ok5 := tagInterface.(string); ok5 && productTag == tag {
						hasTag = true
						break
					}
				}
			}
		}

		if hasTag {
			summary := map[string]any{
				"name":     productMap["name"],
				"label":    productMap["label"],
				"category": productMap["category"],
				"uri":      fmt.Sprintf("%s/products/%s", cm.baseURL, productMap["name"]),
			}

			if aliases, ok3 := productMap["aliases"]; ok3 {
				summary["aliases"] = aliases
			}

			if tags, ok3 := productMap["tags"]; ok3 {
				summary["tags"] = tags
			}

			products = append(products, summary)
		}
	}

	productsResponse := map[string]any{
		"schema_version": fullResponse["schema_version"],
		"total":          len(products),
		"result":         products,
	}

	productsJSON, err := json.Marshal(productsResponse)
	if err != nil {
		return
	}

	return productsJSON, true
}

// ensureCacheDir creates the cache directory if it doesn't exist.
func (cm *CacheManager) ensureCacheDir() error {
	return os.MkdirAll(cm.baseDir, dirPerm)
}

// generateCacheKey creates a cache key from endpoint and parameters.
func (cm *CacheManager) generateCacheKey(endpoint string, params ...string) string {
	endpoint = strings.TrimPrefix(endpoint, "/")
	endpoint = cmp.Or(strings.ReplaceAll(endpoint, "/", "-"), "index")

	if len(params) == 0 {
		return endpoint + cacheExt
	}

	paramStr := strings.Join(params, "|")
	hash := fmt.Sprintf("%x", md5.Sum([]byte(paramStr))) //nolint:gosec // fine for cache keys

	return fmt.Sprintf("%s-%s%s", endpoint, hash[:8], cacheExt)
}

// getCacheFilePath returns the full path to a cache file.
func (cm *CacheManager) getCacheFilePath(key string) string {
	return filepath.Join(cm.baseDir, key)
}

// isFullEndpoint checks if this is a --full products endpoint.
func (cm *CacheManager) isFullEndpoint(endpoint string) bool {
	return endpoint == "/products/full" || endpoint == "products/full"
}
