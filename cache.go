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

// CacheManager handles caching of API responses.
type CacheManager struct {
	baseDir    string
	enabled    bool
	defaultTTL time.Duration
	fullTTL    time.Duration
}

const (
	fullTTL  = 24 * time.Hour // The TTL used for full endpoints (e.g., /products/full).
	cacheExt = ".eol_cache.json"
)

var errRefusingToClear = errors.New("refusing to clear")

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
			baseDir = filepath.Join(homeDir, ".cache", "eol")
		}
	}

	return &CacheManager{
		baseDir:    baseDir,
		enabled:    enabled,
		defaultTTL: defaultTTL,
		fullTTL:    fullTTL,
	}
}

// Get retrieves data from cache if valid.
func (cm *CacheManager) Get(endpoint string, params ...string) (_ json.RawMessage, found bool) {
	// For --full endpoints, always check cache regardless of enabled flag.
	if !cm.enabled && !cm.isFullEndpoint(endpoint) {
		return
	}

	key := cm.generateCacheKey(endpoint, params...)
	filePath := cm.getCacheFilePath(key)

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

// GetReleaseFromProductCache attempts to find a specific release in cached product data.
// Returns the release data and true if found, nil and false if not found or not cached.
//
//nolint:gocognit // ok
func (cm *CacheManager) GetReleaseFromProductCache(product, release string) (_ json.RawMessage, found bool) {
	if !cm.enabled {
		return
	}

	productEndpoint := "/products/" + product

	productCache, ok := cm.Get(productEndpoint, product)
	if !ok {
		return
	}

	fullProductResponse := map[string]any{}
	if err := json.Unmarshal(productCache, &fullProductResponse); err != nil {
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
		var (
			releaseMap map[string]any
			name       string
		)

		releaseMap, ok = r.(map[string]any)
		if !ok {
			continue
		}

		name, ok = releaseMap["name"].(string)
		if !ok {
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

// GetProductFromFullCache attempts to find a specific product in cached ProductsFull data.
// Returns the product data and true if found, nil and false if not found or not cached.
func (cm *CacheManager) GetProductFromFullCache(product string) (_ json.RawMessage, found bool) {
	if !cm.enabled {
		return
	}

	// Look for cached ProductsFull data.
	fullCache, ok := cm.Get("/products/full")
	if !ok {
		return
	}

	fullResponse := map[string]any{}
	if err := json.Unmarshal(fullCache, &fullResponse); err != nil {
		return
	}

	result, found := fullResponse["result"].([]any)
	if !found {
		return
	}

	for _, p := range result {
		var (
			productMap map[string]any
			name       string
		)

		productMap, ok = p.(map[string]any)
		if !ok {
			continue
		}

		name, ok = productMap["name"].(string)
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
				return
			}

			return productJSON, true
		}
	}

	return
}

// GetProductsFromFullCache attempts to extract a basic products list from cached ProductsFull data.
// Returns the products data and true if found, nil and false if not found or not cached.
func (cm *CacheManager) GetProductsFromFullCache() (_ json.RawMessage, found bool) {
	if !cm.enabled {
		return
	}

	fullCache, ok := cm.Get("/products/full")
	if !ok {
		return
	}

	fullResponse := map[string]any{}
	if err := json.Unmarshal(fullCache, &fullResponse); err != nil {
		return
	}

	result, found := fullResponse["result"].([]any)
	if !found {
		return
	}

	products := make([]map[string]any, 0, len(result))

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
		return
	}

	return productsJSON, true
}

// GetReleaseFromFullCache attempts to find a specific release in cached ProductsFull data.
// This is similar to GetReleaseFromProductCache but uses the full products cache instead.
//
//nolint:gocognit,gocyclo,cyclop,funlen // ok
func (cm *CacheManager) GetReleaseFromFullCache(product, release string) (_ json.RawMessage, found bool) {
	if !cm.enabled {
		return
	}

	fullCache, ok := cm.Get("/products/full")
	if !ok {
		return
	}

	fullResponse := map[string]any{}
	if err := json.Unmarshal(fullCache, &fullResponse); err != nil {
		return
	}

	result, found := fullResponse["result"].([]any)
	if !found {
		return
	}

	for _, p := range result {
		var (
			productMap map[string]any
			name       string
			releases   []any
		)

		productMap, ok = p.(map[string]any)
		if !ok {
			continue
		}

		name, ok = productMap["name"].(string)
		if !ok || name != product {
			continue
		}

		releases, ok = productMap["releases"].([]any)
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
