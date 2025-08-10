package eol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewCacheManager(t *testing.T) {
	t.Parallel()

	// Test with custom directory.
	cm := NewCacheManager("/tmp/test-cache", true, 2*time.Hour)
	if cm.baseDir != "/tmp/test-cache" {
		t.Errorf("Expected baseDir '/tmp/test-cache', got '%s'", cm.baseDir)
	}

	if !cm.enabled {
		t.Error("Expected cache to be enabled")
	}

	if cm.defaultTTL != 2*time.Hour {
		t.Errorf("Expected defaultTTL 2h, got %v", cm.defaultTTL)
	}

	if cm.fullTTL != 24*time.Hour {
		t.Errorf("Expected fullTTL 24h, got %v", cm.fullTTL)
	}

	// Test with empty directory (should use default).
	cm2 := NewCacheManager("", false, time.Hour)
	if cm2.baseDir == "" {
		t.Error("Expected baseDir to be set to default")
	}

	if cm2.enabled {
		t.Error("Expected cache to be disabled")
	}
}

func TestCacheManagerSetEnabled(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager("/tmp/test", false, time.Hour)
	if cm.enabled {
		t.Error("Expected cache to be disabled initially")
	}

	cm.SetEnabled(true)

	if !cm.enabled {
		t.Error("Expected cache to be enabled after SetEnabled(true)")
	}
}

func TestCacheManagerSetDefaultTTL(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager("/tmp/test", true, time.Hour)
	if cm.defaultTTL != time.Hour {
		t.Errorf("Expected initial TTL 1h, got %v", cm.defaultTTL)
	}

	cm.SetDefaultTTL(30 * time.Minute)

	if cm.defaultTTL != 30*time.Minute {
		t.Errorf("Expected TTL 30m, got %v", cm.defaultTTL)
	}
}

func TestCacheManagerGenerateCacheKey(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager("/tmp/test", true, time.Hour)

	//nolint:govet // ok
	tests := []struct {
		endpoint string
		params   []string
		expected string
	}{
		{"products", nil, "products.json"},
		{"/products", nil, "products.json"},
		{"products/full", nil, "products-full.json"},
		{"/products/ubuntu", nil, "products-ubuntu.json"},
		{"products", []string{"param1"}, "products-a2cbb63a.json"},
		{"products", []string{"param1"}, "products-a2cbb63a.json"}, // MD5 hash of "param1".
	}

	for _, test := range tests {
		result := cm.generateCacheKey(test.endpoint, test.params...)
		if result != test.expected {
			t.Errorf("generateCacheKey(%s, %v) = %s, expected %s", test.endpoint, test.params, result, test.expected)
		}
	}
}

func TestCacheManagerIsFullEndpoint(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager("/tmp/test", true, time.Hour)

	tests := []struct {
		endpoint string
		expected bool
	}{
		{"/products/full", true},
		{"products/full", true},
		{"/products", false},
		{"products", false},
		{"/categories", false},
	}

	for _, test := range tests {
		result := cm.isFullEndpoint(test.endpoint)
		if result != test.expected {
			t.Errorf("isFullEndpoint(%s) = %t, expected %t", test.endpoint, result, test.expected)
		}
	}
}

//nolint:paralleltest // TempDir
func TestCacheManagerSetAndGet(t *testing.T) {
	cm := NewCacheManager(t.TempDir(), true, time.Hour)
	testData := map[string]any{
		"test": "value",
		"num":  42,
	}

	err := cm.Set("test-endpoint", testData)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	cached, found := cm.Get("test-endpoint")
	if !found {
		t.Fatal("Expected to find cached data")
	}

	var cachedMap map[string]any

	if err = json.Unmarshal(cached, &cachedMap); err != nil {
		t.Fatalf("Failed to unmarshal cached data: %v", err)
	}

	if cachedMap["test"] != "value" {
		t.Errorf("Expected test='value', got %v", cachedMap["test"])
	}

	if cachedMap["num"] != float64(42) {
		t.Errorf("Expected num=42, got %v", cachedMap["num"])
	}
}

//nolint:paralleltest // TempDir
func TestCacheManagerGetNonExistent(t *testing.T) {
	cm := NewCacheManager(t.TempDir(), true, time.Hour)

	_, found := cm.Get("non-existent")
	if found {
		t.Error("Expected not to find non-existent cache")
	}
}

//nolint:paralleltest // TempDir
func TestCacheManagerExpiredCache(t *testing.T) {
	cm := NewCacheManager(t.TempDir(), true, 10*time.Millisecond)
	testData := map[string]any{"test": "value"}

	err := cm.Set("test-endpoint", testData)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	_, found := cm.Get("test-endpoint")
	if found {
		t.Error("Expected not to find expired cache")
	}
}

//nolint:paralleltest // TempDir
func TestCacheManagerDisabledCache(t *testing.T) {
	cm := NewCacheManager(t.TempDir(), false, time.Hour)
	testData := map[string]any{"test": "value"}

	err := cm.Set("test-endpoint", testData)
	if err != nil {
		t.Fatalf("Set should not error when disabled: %v", err)
	}

	_, found := cm.Get("test-endpoint")
	if found {
		t.Error("Expected not to find cache when disabled")
	}
}

//nolint:paralleltest // TempDir
func TestCacheManagerFullEndpointAlwaysCached(t *testing.T) {
	cm := NewCacheManager(t.TempDir(), false, time.Hour)
	testData := map[string]any{"test": "full"}

	err := cm.Set("products/full", testData)
	if err != nil {
		t.Fatalf("Failed to set cache for full endpoint: %v", err)
	}

	cached, found := cm.Get("products/full")
	if !found {
		t.Error("Expected to find cache for full endpoint even when disabled")
	}

	var cachedMap map[string]any

	if err = json.Unmarshal(cached, &cachedMap); err != nil {
		t.Fatalf("Failed to unmarshal cached data: %v", err)
	}

	if cachedMap["test"] != "full" {
		t.Errorf("Expected test='full', got %v", cachedMap["test"])
	}
}

//nolint:paralleltest // TempDir
func TestCacheManagerClear(t *testing.T) {
	cm := NewCacheManager(t.TempDir(), true, time.Hour)
	testData := map[string]any{"test": "value"}

	err := cm.Set("test-endpoint", testData)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	_, found := cm.Get("test-endpoint")
	if !found {
		t.Fatal("Expected to find cached data before clear")
	}

	err = cm.Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	_, found = cm.Get("test-endpoint")
	if found {
		t.Error("Expected not to find cache after clear")
	}
}

//nolint:paralleltest // TempDir
func TestCacheManagerClearExpired(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, true, time.Hour)
	validData := map[string]any{"valid": "data"}

	err := cm.Set("valid-endpoint", validData)
	if err != nil {
		t.Fatalf("Failed to set valid cache: %v", err)
	}

	expiredEntry := CacheEntry{
		Timestamp: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
		Data:      json.RawMessage(`{"expired": true}`),
		Endpoint:  "expired-endpoint",
	}

	expiredJSON, err := json.MarshalIndent(expiredEntry, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal expired entry: %v", err)
	}

	expiredFile := filepath.Join(tempDir, "expired-endpoint.json")

	err = os.WriteFile(expiredFile, expiredJSON, 0o644)
	if err != nil {
		t.Fatalf("Failed to write expired cache file: %v", err)
	}

	err = cm.ClearExpired()
	if err != nil {
		t.Fatalf("Failed to clear expired: %v", err)
	}

	_, found := cm.Get("valid-endpoint")
	if !found {
		t.Error("Expected valid cache to still exist")
	}

	if _, err = os.Stat(expiredFile); !os.IsNotExist(err) {
		t.Error("Expected expired cache file to be removed")
	}
}

func TestNewCacheManagerComprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expectedResult func(*CacheManager) bool
		name           string
		baseDir        string
		defaultTTL     time.Duration
		enabled        bool
	}{
		{
			name:       "custom base directory",
			baseDir:    "/custom/cache/dir",
			enabled:    true,
			defaultTTL: time.Hour,
			expectedResult: func(cm *CacheManager) bool {
				return cm.baseDir == "/custom/cache/dir" &&
					cm.enabled == true &&
					cm.defaultTTL == time.Hour &&
					cm.fullTTL == 24*time.Hour
			},
		},
		{
			name:       "disabled cache",
			baseDir:    "",
			enabled:    false,
			defaultTTL: 30 * time.Minute,
			expectedResult: func(cm *CacheManager) bool {
				return !cm.enabled &&
					cm.defaultTTL == 30*time.Minute &&
					cm.fullTTL == 24*time.Hour &&
					cm.baseDir != ""
			},
		},
		{
			name:       "zero TTL",
			baseDir:    "/tmp/cache",
			enabled:    true,
			defaultTTL: 0,
			expectedResult: func(cm *CacheManager) bool {
				return cm.baseDir == "/tmp/cache" &&
					cm.enabled == true &&
					cm.defaultTTL == 0 &&
					cm.fullTTL == 24*time.Hour
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := NewCacheManager(tt.baseDir, tt.enabled, tt.defaultTTL)

			if cm == nil {
				t.Fatal("NewCacheManager returned nil")
			}

			if !tt.expectedResult(cm) {
				t.Errorf("CacheManager validation failed for test case: %s", tt.name)
				t.Errorf("  baseDir: %s", cm.baseDir)
				t.Errorf("  enabled: %v", cm.enabled)
				t.Errorf("  defaultTTL: %v", cm.defaultTTL)
				t.Errorf("  fullTTL: %v", cm.fullTTL)
			}
		})
	}
}

func TestNewCacheManagerDefaultPaths(t *testing.T) {
	t.Parallel()

	// Test that empty baseDir results in a non-empty path.
	cm := NewCacheManager("", true, time.Hour)

	if cm.baseDir == "" {
		t.Error("baseDir should not be empty when default path is used")
	}

	// The exact path depends on the OS and user's home directory,
	// but it should contain some expected patterns.
	expectedPatterns := []string{"eol", "cache"}

	found := false

	for _, pattern := range expectedPatterns {
		if strings.Contains(strings.ToLower(cm.baseDir), pattern) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("baseDir %q should contain one of %v", cm.baseDir, expectedPatterns)
	}
}

//nolint:paralleltest // TempDir
func TestCacheManagerGetStats(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, true, time.Hour)
	testData := map[string]any{"test": "value"}

	err := cm.Set("test-endpoint", testData)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	stats, err := cm.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Enabled != true {
		t.Errorf("Expected enabled=true, got %v", stats.Enabled)
	}

	if stats.CacheDir != tempDir {
		t.Errorf("Expected cache_dir=%s, got %v", tempDir, stats.CacheDir)
	}

	if stats.TotalFiles < 1 {
		t.Errorf("Expected at least 1 total file, got %v", stats.TotalFiles)
	}

	if stats.ValidFiles < 1 {
		t.Errorf("Expected at least 1 valid file, got %v", stats.ValidFiles)
	}
}

func TestCacheManagerMustUseCache(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager("/tmp/test", true, time.Hour)
	tests := []struct {
		endpoint string
		expected bool
	}{
		{"/products/full", true},
		{"products/full", true},
		{"/products", false},
		{"products", false},
	}

	for _, test := range tests {
		result := cm.MustUseCache(test.endpoint)
		if result != test.expected {
			t.Errorf("MustUseCache(%s) = %t, expected %t", test.endpoint, result, test.expected)
		}
	}
}

func TestCacheManagerGetReleaseFromProductCache(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager(t.TempDir(), true, time.Hour)
	productData := map[string]any{
		"schema_version": "1.2.0",
		"last_modified":  "2025-01-11T00:00:00Z",
		"result": map[string]any{
			"name":     "go",
			"label":    "Go",
			"category": "lang",
			"releases": []any{
				map[string]any{
					"name":         "1.24",
					"label":        "1.24",
					"releaseDate":  "2025-02-11",
					"isLts":        false,
					"isMaintained": true,
					"isEol":        false,
					"latest": map[string]any{
						"name": "1.24.0",
						"date": "2025-02-11",
					},
				},
				map[string]any{
					"name":         "1.23",
					"label":        "1.23",
					"releaseDate":  "2024-08-13",
					"isLts":        false,
					"isMaintained": true,
					"isEol":        false,
					"latest": map[string]any{
						"name": "1.23.4",
						"date": "2024-12-03",
					},
				},
			},
		},
	}

	if err := cm.Set("/products/go", productData, "go"); err != nil {
		t.Fatalf("Failed to cache product data: %v", err)
	}

	//nolint:govet // ok
	tests := []struct {
		name            string
		product         string
		release         string
		expectedFound   bool
		expectedName    string
		expectedVersion string
	}{
		{
			name:            "existing release",
			product:         "go",
			release:         "1.24",
			expectedFound:   true,
			expectedName:    "1.24",
			expectedVersion: "1.2.0",
		},
		{
			name:            "another existing release",
			product:         "go",
			release:         "1.23",
			expectedFound:   true,
			expectedName:    "1.23",
			expectedVersion: "1.2.0",
		},
		{
			name:          "non-existent release",
			product:       "go",
			release:       "1.22",
			expectedFound: false,
		},
		{
			name:          "non-existent product",
			product:       "nonexistent",
			release:       "1.0",
			expectedFound: false,
		},
		{
			name:            "version normalization",
			product:         "go",
			release:         "1.23.4", // Should normalize to "1.23".
			expectedFound:   true,
			expectedName:    "1.23",
			expectedVersion: "1.2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			releaseData, found := cm.GetReleaseFromProductCache(tt.product, tt.release)

			if found != tt.expectedFound {
				t.Errorf("Expected found=%v, got found=%v", tt.expectedFound, found)
				return
			}

			if !tt.expectedFound {
				return // Test passed - we expected not to find it.
			}

			// Parse the returned JSON to verify structure.
			var releaseResponse map[string]any

			err := json.Unmarshal(releaseData, &releaseResponse)
			if err != nil {
				t.Fatalf("Failed to unmarshal release response: %v", err)
			}

			if schema, ok := releaseResponse["schema_version"].(string); !ok || schema != tt.expectedVersion {
				t.Errorf("Expected schema_version=%s, got %v", tt.expectedVersion, releaseResponse["schema_version"])
			}

			result, ok := releaseResponse["result"].(map[string]any)
			if !ok {
				t.Fatalf("Expected result to be a map, got %T", releaseResponse["result"])
			}

			if name, ok := result["name"].(string); !ok || name != tt.expectedName { //nolint:govet // ok
				t.Errorf("Expected release name=%s, got %v", tt.expectedName, result["name"])
			}
		})
	}
}

func TestCacheManagerGetReleaseFromProductCacheCacheDisabled(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager(t.TempDir(), false, time.Hour) // Cache disabled.

	_, found := cm.GetReleaseFromProductCache("go", "1.24")
	if found {
		t.Error("Expected not to find release when cache is disabled")
	}
}

func TestNewCacheManagerEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cacheDir string
		enabled  bool
		ttl      time.Duration
	}{
		{
			name:     "empty cache dir",
			cacheDir: "",
			enabled:  true,
			ttl:      time.Hour,
		},
		{
			name:     "zero TTL",
			cacheDir: t.TempDir(),
			enabled:  true,
			ttl:      0,
		},
		{
			name:     "disabled cache",
			cacheDir: t.TempDir(),
			enabled:  false,
			ttl:      time.Hour,
		},
	}

	for _, tt := range tests {
		//nolint:staticcheck // ok
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := NewCacheManager(tt.cacheDir, tt.enabled, tt.ttl)
			if cm == nil {
				t.Error("Expected non-nil cache manager")
			}

			if cm.enabled != tt.enabled {
				t.Errorf("Expected enabled=%v, got %v", tt.enabled, cm.enabled)
			}

			if cm.defaultTTL != tt.ttl {
				t.Errorf("Expected TTL=%v, got %v", tt.ttl, cm.defaultTTL)
			}
		})
	}
}

func TestCacheManagerGetProductFromFullCache(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager(t.TempDir(), true, time.Hour)

	// Test with no cached data.
	_, found := cm.GetProductFromFullCache("go")
	if found {
		t.Error("Expected not to find product when no cache exists")
	}

	// Test with disabled cache.
	cmDisabled := NewCacheManager(t.TempDir(), false, time.Hour)

	_, found = cmDisabled.GetProductFromFullCache("go")
	if found {
		t.Error("Expected not to find product when cache is disabled")
	}
}

func TestCacheManagerGetReleaseFromFullCache(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager(t.TempDir(), true, time.Hour)

	// Test with no cached data.
	_, found := cm.GetReleaseFromFullCache("go", "1.24")
	if found {
		t.Error("Expected not to find release when no cache exists")
	}

	// Test with disabled cache.
	cmDisabled := NewCacheManager(t.TempDir(), false, time.Hour)

	_, found = cmDisabled.GetReleaseFromFullCache("go", "1.24")
	if found {
		t.Error("Expected not to find release when cache is disabled")
	}
}

func TestCacheManagerGetProductsFromFullCache(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager(t.TempDir(), true, time.Hour)

	// Test with no cached data.
	_, found := cm.GetProductsFromFullCache()
	if found {
		t.Error("Expected not to find products when no cache exists")
	}

	// Test with disabled cache.
	cmDisabled := NewCacheManager(t.TempDir(), false, time.Hour)

	_, found = cmDisabled.GetProductsFromFullCache()
	if found {
		t.Error("Expected not to find products when cache is disabled")
	}
}

//nolint:staticcheck // ok
func TestNewCacheManagerDefaultPath(t *testing.T) {
	t.Parallel()

	// Test with empty cache dir to trigger default path logic.
	cm := NewCacheManager("", true, time.Hour)
	if cm == nil {
		t.Error("Expected non-nil cache manager")
	}

	if cm.baseDir == "" {
		t.Error("Expected non-empty base directory")
	}

	// Verify it creates the directory structure.
	if cm.enabled && cm.baseDir != "" {
		// The cache manager should handle directory creation internally.
		stats, err := cm.GetStats()
		if err != nil {
			t.Errorf("GetStats should work even with default path: %v", err)
		}

		if stats.TotalFiles < 0 {
			t.Error("Total files should not be negative")
		}
	}
}

func TestNewCacheManagerComprehensiveEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testFunc func(*testing.T, *CacheManager)
		name     string
		cacheDir string
		ttl      time.Duration
		enabled  bool
	}{
		{
			name:     "empty cache dir with user home",
			cacheDir: "",
			enabled:  true,
			ttl:      time.Hour,
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				// Should use default cache path.
				if cm.baseDir == "" {
					t.Error("Expected non-empty base directory for default path")
				}
			},
		},
		{
			name:     "disabled cache manager",
			cacheDir: t.TempDir(),
			enabled:  false,
			ttl:      time.Hour,
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				// Test that disabled cache doesn't create directories.
				_, found := cm.Get("/test")
				if found {
					t.Error("Disabled cache should not return data")
				}
			},
		},
		{
			name:     "zero TTL",
			cacheDir: t.TempDir(),
			enabled:  true,
			ttl:      0,
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				if cm.defaultTTL != 0 {
					t.Errorf("Expected TTL to be 0, got %v", cm.defaultTTL)
				}
			},
		},
		{
			name:     "negative TTL",
			cacheDir: t.TempDir(),
			enabled:  true,
			ttl:      -time.Hour,
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				if cm.defaultTTL != -time.Hour {
					t.Errorf("Expected TTL to be -1h, got %v", cm.defaultTTL)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := NewCacheManager(tt.cacheDir, tt.enabled, tt.ttl)
			if cm == nil {
				t.Error("Expected non-nil cache manager")
				return
			}

			if cm.enabled != tt.enabled {
				t.Errorf("Expected enabled=%v, got %v", tt.enabled, cm.enabled)
			}

			if cm.defaultTTL != tt.ttl {
				t.Errorf("Expected TTL=%v, got %v", tt.ttl, cm.defaultTTL)
			}

			if tt.testFunc != nil {
				tt.testFunc(t, cm)
			}
		})
	}
}

func TestCacheManagerInvalidJSON(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager(t.TempDir(), true, time.Hour)

	// Set invalid JSON data.
	invalidJSON := []byte("{invalid json")

	err := cm.Set("/test", invalidJSON, "test")
	if err != nil {
		t.Fatalf("Failed to set invalid JSON: %v", err)
	}

	// GetProductFromFullCache should handle invalid JSON gracefully.
	_, found := cm.GetProductFromFullCache("test")
	if found {
		t.Error("Expected not to find product with invalid JSON")
	}

	// GetReleaseFromFullCache should handle invalid JSON gracefully.
	_, found = cm.GetReleaseFromFullCache("test", "1.0")
	if found {
		t.Error("Expected not to find release with invalid JSON")
	}
}

func TestCacheManagerGetStatsEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup    func(*testing.T) *CacheManager
		testFunc func(*testing.T, *CacheManager)
		name     string
	}{
		{
			name: "disabled cache",
			setup: func(t *testing.T) *CacheManager {
				t.Helper()

				return NewCacheManager(t.TempDir(), false, time.Hour)
			},
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				stats, err := cm.GetStats()
				if err != nil {
					t.Errorf("GetStats should work for disabled cache: %v", err)
				}
				if stats.TotalFiles != 0 {
					t.Errorf("Expected 0 files for disabled cache, got %d", stats.TotalFiles)
				}
			},
		},
		{
			name: "empty cache directory",
			setup: func(t *testing.T) *CacheManager {
				t.Helper()

				return NewCacheManager(t.TempDir(), true, time.Hour)
			},
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				stats, err := cm.GetStats()
				if err != nil {
					t.Errorf("GetStats should work for empty cache: %v", err)
				}
				if stats.TotalFiles != 0 {
					t.Errorf("Expected 0 files for empty cache, got %d", stats.TotalFiles)
				}
				if stats.TotalSize != 0 {
					t.Errorf("Expected 0 size for empty cache, got %d", stats.TotalSize)
				}
			},
		},
		{
			name: "cache with some files",
			setup: func(t *testing.T) *CacheManager {
				t.Helper()

				cm := NewCacheManager(t.TempDir(), true, time.Hour)
				// Add some cache entries.
				cm.Set("/test1", []byte("test data 1"), "test1")
				cm.Set("/test2", []byte("test data 2"), "test2")

				return cm
			},
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				stats, err := cm.GetStats()
				if err != nil {
					t.Errorf("GetStats failed: %v", err)
				}
				if stats.TotalFiles < 2 {
					t.Errorf("Expected at least 2 files, got %d", stats.TotalFiles)
				}
				if stats.TotalSize == 0 {
					t.Error("Expected non-zero size for cache with files")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := tt.setup(t)
			tt.testFunc(t, cm)
		})
	}
}

func TestCacheManagerClearExpiredEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup    func(*testing.T) *CacheManager
		testFunc func(*testing.T, *CacheManager)
		name     string
	}{
		{
			name: "disabled cache",
			setup: func(t *testing.T) *CacheManager {
				t.Helper()

				return NewCacheManager(t.TempDir(), false, time.Hour)
			},
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				err := cm.ClearExpired()
				if err != nil {
					t.Errorf("ClearExpired should work for disabled cache: %v", err)
				}
			},
		},
		{
			name: "empty cache directory",
			setup: func(t *testing.T) *CacheManager {
				t.Helper()

				return NewCacheManager(t.TempDir(), true, time.Hour)
			},
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				err := cm.ClearExpired()
				if err != nil {
					t.Errorf("ClearExpired should work for empty cache: %v", err)
				}
			},
		},
		{
			name: "cache with expired entries",
			setup: func(t *testing.T) *CacheManager {
				t.Helper()

				cm := NewCacheManager(t.TempDir(), true, -time.Hour) // Negative TTL makes everything expire.
				cm.Set("/test1", []byte("test data 1"), "test1")
				cm.Set("/test2", []byte("test data 2"), "test2")

				return cm
			},
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				err := cm.ClearExpired()
				if err != nil {
					t.Errorf("ClearExpired failed: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := tt.setup(t)
			tt.testFunc(t, cm)
		})
	}
}
