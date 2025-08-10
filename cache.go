package eol

import (
	"crypto/md5" //nolint:gosec // MD5 is fine for cache keys
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

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
	TotalSize    int64  `json:"total_size"`
	TotalFiles   int    `json:"total_files"`
	ExpiredFiles int    `json:"expired_files"`
	ValidFiles   int    `json:"valid_files"`
	Enabled      bool   `json:"enabled"`
}

// CacheManager handles caching of API responses.
type CacheManager struct {
	baseDir    string
	enabled    bool
	defaultTTL time.Duration
	fullTTL    time.Duration // Special TTL for --full endpoints (24h).
}

// NewCacheManager creates a new cache manager.
func NewCacheManager(baseDir string, enabled bool, defaultTTL time.Duration) *CacheManager {
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
			baseDir = filepath.Join(homeDir, ".local", "state", "eol")
		}
	}

	return &CacheManager{
		baseDir:    baseDir,
		enabled:    enabled,
		defaultTTL: defaultTTL,
		fullTTL:    24 * time.Hour, //nolint:mnd // full always cached for 24h.
	}
}

// SetEnabled enables or disables caching.
func (cm *CacheManager) SetEnabled(enabled bool) {
	cm.enabled = enabled
}

// SetDefaultTTL sets the default cache TTL.
func (cm *CacheManager) SetDefaultTTL(ttl time.Duration) {
	cm.defaultTTL = ttl
}

// Get retrieves data from cache if valid.
func (cm *CacheManager) Get(endpoint string, params ...string) (json.RawMessage, bool) {
	// For --full endpoints, always check cache regardless of enabled flag.
	if !cm.enabled && !cm.isFullEndpoint(endpoint) {
		return nil, false
	}

	key := cm.generateCacheKey(endpoint, params...)
	filePath := cm.getCacheFilePath(key)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, false
	}

	data, err := os.ReadFile(filePath) //nolint:gosec // Reading cache file is safe
	if err != nil {
		return nil, false
	}

	var entry CacheEntry

	if err = json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		os.Remove(filePath) //nolint:errcheck,gosec // TODO
		return nil, false
	}

	return entry.Data, true
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

	if err = os.WriteFile(filePath, jsonData, 0o640); err != nil { //nolint:mnd // it's a file permission
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return
}

// Clear removes all cache files.
func (cm *CacheManager) Clear() error {
	return os.RemoveAll(cm.baseDir)
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(cm.baseDir, entry.Name())

		var data []byte

		//nolint:gosec // Reading cache file is safe
		if data, err = os.ReadFile(filePath); err != nil {
			continue
		}

		var cacheEntry CacheEntry

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
		Enabled:      cm.enabled,
		CacheDir:     cm.baseDir,
		DefaultTTL:   cm.defaultTTL.String(),
		FullTTL:      cm.fullTTL.String(),
		TotalFiles:   0,
		TotalSize:    int64(0),
		ExpiredFiles: 0,
		ValidFiles:   0,
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
		stats.TotalSize += fileInfo.Size()

		var data []byte

		//nolint:gosec // Reading cache file is safe
		if data, err = os.ReadFile(filePath); err != nil {
			continue
		}

		var cacheEntry CacheEntry

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

// GetReleaseFromProductCache attempts to find a specific release in cached product data.
// Returns the release data and true if found, nil and false if not found or not cached.
//
//nolint:gocognit // ok
func (cm *CacheManager) GetReleaseFromProductCache(product, release string) (json.RawMessage, bool) {
	if !cm.enabled {
		return nil, false
	}

	productEndpoint := "/products/" + product

	productCache, found := cm.Get(productEndpoint, product)
	if !found {
		return nil, false
	}

	var fullProductResponse map[string]any

	if err := json.Unmarshal(productCache, &fullProductResponse); err != nil {
		return nil, false
	}

	result, ok := fullProductResponse["result"].(map[string]any)
	if !ok {
		return nil, false
	}

	releases, ok := result["releases"].([]any)
	if !ok {
		return nil, false
	}

	for _, r := range releases {
		releaseMap, ok := r.(map[string]any) //nolint:govet // ok
		if !ok {
			continue
		}

		name, ok := releaseMap["name"].(string)
		if !ok {
			continue
		}

		// Check both original and normalized release names.
		if name == release || name == normalizeVersion(release) {
			// Create a ProductReleaseResponse format.
			releaseResponse := map[string]any{
				"schema_version": fullProductResponse["schema_version"],
				"result":         releaseMap,
			}

			releaseJSON, err := json.Marshal(releaseResponse)
			if err != nil {
				return nil, false
			}

			return releaseJSON, true
		}
	}

	return nil, false
}

// GetProductFromFullCache attempts to find a specific product in cached ProductsFull data.
// Returns the product data and true if found, nil and false if not found or not cached.
func (cm *CacheManager) GetProductFromFullCache(product string) (json.RawMessage, bool) {
	if !cm.enabled {
		return nil, false
	}

	// Look for cached ProductsFull data.
	fullCache, found := cm.Get("/products/full")
	if !found {
		return nil, false
	}

	var fullResponse map[string]any

	if err := json.Unmarshal(fullCache, &fullResponse); err != nil {
		return nil, false
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return nil, false
	}

	for _, p := range result {
		productMap, ok := p.(map[string]any) //nolint:govet // ok
		if !ok {
			continue
		}

		name, ok := productMap["name"].(string)
		if !ok {
			continue
		}

		if name == product {
			productResponse := map[string]any{
				"schema_version": fullResponse["schema_version"],
				"last_modified":  "2025-01-11T00:00:00Z", // Use a reasonable default.
				"result":         productMap,
			}

			productJSON, err := json.Marshal(productResponse)
			if err != nil {
				return nil, false
			}

			return productJSON, true
		}
	}

	return nil, false
}

// GetProductsFromFullCache attempts to extract a basic products list from cached ProductsFull data.
// Returns the products data and true if found, nil and false if not found or not cached.
func (cm *CacheManager) GetProductsFromFullCache() (json.RawMessage, bool) {
	if !cm.enabled {
		return nil, false
	}

	fullCache, found := cm.Get("/products/full")
	if !found {
		return nil, false
	}

	var fullResponse map[string]any

	if err := json.Unmarshal(fullCache, &fullResponse); err != nil {
		return nil, false
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return nil, false
	}

	var products []map[string]any //nolint:prealloc // ok

	for _, p := range result {
		productMap, ok := p.(map[string]any) //nolint:govet // ok
		if !ok {
			continue
		}

		summary := map[string]any{
			"name":     productMap["name"],
			"label":    productMap["label"],
			"category": productMap["category"],
			"uri":      fmt.Sprintf("https://endoflife.date/api/v1/products/%s", productMap["name"]),
		}

		if aliases, ok := productMap["aliases"]; ok { //nolint:govet // ok
			summary["aliases"] = aliases
		}

		if tags, ok := productMap["tags"]; ok { //nolint:govet // ok
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
		return nil, false
	}

	return productsJSON, true
}

// GetReleaseFromFullCache attempts to find a specific release in cached ProductsFull data.
// This is similar to GetReleaseFromProductCache but uses the full products cache instead.
//
//nolint:gocognit,gocyclo,cyclop,funlen // ok
func (cm *CacheManager) GetReleaseFromFullCache(product, release string) (json.RawMessage, bool) {
	if !cm.enabled {
		return nil, false
	}

	fullCache, found := cm.Get("/products/full")
	if !found {
		return nil, false
	}

	var fullResponse map[string]any

	if err := json.Unmarshal(fullCache, &fullResponse); err != nil {
		return nil, false
	}

	result, ok := fullResponse["result"].([]any)
	if !ok {
		return nil, false
	}

	for _, p := range result {
		productMap, ok := p.(map[string]any) //nolint:govet // ok
		if !ok {
			continue
		}

		name, ok := productMap["name"].(string)
		if !ok || name != product {
			continue
		}

		releases, ok := productMap["releases"].([]any)
		if !ok {
			continue
		}

		for _, r := range releases {
			releaseMap, ok := r.(map[string]any) //nolint:govet // ok
			if !ok {
				continue
			}

			releaseName, ok := releaseMap["name"].(string)
			if !ok {
				continue
			}

			// Check both original and normalized release names.
			if releaseName == release || releaseName == normalizeVersion(release) {
				releaseResponse := map[string]any{
					"schema_version": fullResponse["schema_version"],
					"result":         releaseMap,
				}

				releaseJSON, err := json.Marshal(releaseResponse)
				if err != nil {
					return nil, false
				}

				return releaseJSON, true
			}
		}

		return nil, false
	}

	return nil, false
}

// ensureCacheDir creates the cache directory if it doesn't exist.
func (cm *CacheManager) ensureCacheDir() error {
	return os.MkdirAll(cm.baseDir, 0o750) //nolint:mnd // it's a file permission
}

// generateCacheKey creates a cache key from endpoint and parameters.
func (cm *CacheManager) generateCacheKey(endpoint string, params ...string) string {
	endpoint = strings.TrimPrefix(endpoint, "/")
	endpoint = strings.ReplaceAll(endpoint, "/", "-")

	if endpoint == "" {
		endpoint = "index"
	}

	if len(params) == 0 {
		return endpoint + ".json"
	}

	paramStr := strings.Join(params, "|")
	hash := fmt.Sprintf("%x", md5.Sum([]byte(paramStr))) //nolint:gosec // MD5 is fine for cache keys

	return fmt.Sprintf("%s-%s.json", endpoint, hash[:8])
}

// getCacheFilePath returns the full path to a cache file.
func (cm *CacheManager) getCacheFilePath(key string) string {
	return filepath.Join(cm.baseDir, key)
}

// isFullEndpoint checks if this is a --full products endpoint.
func (cm *CacheManager) isFullEndpoint(endpoint string) bool {
	return endpoint == "/products/full" || endpoint == "products/full"
}
