package eol

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewCacheManager(t *testing.T) {
	t.Parallel()

	//nolint:govet // ok
	tests := []struct {
		name           string
		baseDir        string
		enabled        bool
		defaultTTL     time.Duration
		expectedResult func(*CacheManager) bool
	}{
		{
			name:       "custom directory enabled",
			baseDir:    "/tmp/test-cache",
			enabled:    true,
			defaultTTL: 2 * time.Hour,
			expectedResult: func(cm *CacheManager) bool {
				return cm.baseDir == "/tmp/test-cache" &&
					cm.enabled == true &&
					cm.defaultTTL == 2*time.Hour &&
					cm.fullTTL == 24*time.Hour
			},
		},
		{
			name:       "empty directory disabled",
			baseDir:    "",
			enabled:    false,
			defaultTTL: time.Hour,
			expectedResult: func(cm *CacheManager) bool {
				return cm.baseDir != "" &&
					cm.enabled == false &&
					cm.defaultTTL == time.Hour &&
					cm.fullTTL == 24*time.Hour
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
		{
			name:       "negative TTL",
			baseDir:    "/tmp/cache",
			enabled:    true,
			defaultTTL: -time.Hour,
			expectedResult: func(cm *CacheManager) bool {
				return cm.baseDir == "/tmp/cache" &&
					cm.enabled == true &&
					cm.defaultTTL == -time.Hour &&
					cm.fullTTL == 24*time.Hour
			},
		},
		{
			name:       "default path validation",
			baseDir:    "",
			enabled:    true,
			defaultTTL: 30 * time.Minute,
			expectedResult: func(cm *CacheManager) bool {
				expectedPatterns := []string{"eol", "cache"}
				found := false

				for _, pattern := range expectedPatterns {
					if strings.Contains(strings.ToLower(cm.baseDir), pattern) {
						found = true
						break
					}
				}

				return cm.baseDir != "" &&
					cm.enabled == true &&
					cm.defaultTTL == 30*time.Minute &&
					cm.fullTTL == 24*time.Hour &&
					found
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := NewCacheManager(tt.baseDir, DefaultBaseURL, tt.enabled, tt.defaultTTL)

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

func TestCacheManagerGenerateCacheKey(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager("/tmp/test", DefaultBaseURL, true, time.Hour)

	//nolint:govet // ok
	tests := []struct {
		name     string
		endpoint string
		params   []string
		expected string
	}{
		{
			name:     "simple endpoint no params",
			endpoint: "products",
			params:   nil,
			expected: "products" + cacheExt,
		},
		{
			name:     "endpoint with leading slash",
			endpoint: "/products",
			params:   nil,
			expected: "products" + cacheExt,
		},
		{
			name:     "full endpoint",
			endpoint: "products/full",
			params:   nil,
			expected: "products-full" + cacheExt,
		},
		{
			name:     "product endpoint",
			endpoint: "/products/ubuntu",
			params:   nil,
			expected: "products-ubuntu" + cacheExt,
		},
		{
			name:     "endpoint with params",
			endpoint: "products",
			params:   []string{"param1"},
			expected: "products-a2cbb63a" + cacheExt,
		},
		{
			name:     "consistent param hashing",
			endpoint: "products",
			params:   []string{"param1"},
			expected: "products-a2cbb63a" + cacheExt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := cm.generateCacheKey(tt.endpoint, tt.params...)
			if result != tt.expected {
				t.Errorf("generateCacheKey(%s, %v) = %s, expected %s", tt.endpoint, tt.params, result, tt.expected)
			}
		})
	}
}

func TestCacheManagerIsFullEndpoint(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager("/tmp/test", DefaultBaseURL, true, time.Hour)

	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "full endpoint with slash",
			endpoint: "/products/full",
			expected: true,
		},
		{
			name:     "full endpoint without slash",
			endpoint: "products/full",
			expected: true,
		},
		{
			name:     "products endpoint with slash",
			endpoint: "/products",
			expected: false,
		},
		{
			name:     "products endpoint without slash",
			endpoint: "products",
			expected: false,
		},
		{
			name:     "categories endpoint",
			endpoint: "/categories",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := cm.isFullEndpoint(tt.endpoint)
			if result != tt.expected {
				t.Errorf("isFullEndpoint(%s) = %t, expected %t", tt.endpoint, result, tt.expected)
			}
		})
	}
}

func TestCacheManagerMustUseCache(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager("/tmp/test", DefaultBaseURL, true, time.Hour)

	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "full endpoint with slash",
			endpoint: "/products/full",
			expected: true,
		},
		{
			name:     "full endpoint without slash",
			endpoint: "products/full",
			expected: true,
		},
		{
			name:     "products endpoint with slash",
			endpoint: "/products",
			expected: false,
		},
		{
			name:     "products endpoint without slash",
			endpoint: "products",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := cm.MustUseCache(tt.endpoint)
			if result != tt.expected {
				t.Errorf("MustUseCache(%s) = %t, expected %t", tt.endpoint, result, tt.expected)
			}
		})
	}
}

//nolint:paralleltest // t.TempDir
func TestCacheManagerSetAndGet(t *testing.T) {
	//nolint:govet // ok
	tests := []struct {
		name           string
		enabled        bool
		endpoint       string
		ttl            time.Duration
		sleep          time.Duration
		testData       map[string]any
		expectedFound  bool
		expectSetError bool
	}{
		{
			name:          "successful set and get",
			enabled:       true,
			endpoint:      "test-endpoint",
			ttl:           time.Hour,
			testData:      map[string]any{"test": "value", "num": 42},
			expectedFound: true,
		},
		{
			name:          "non-existent cache",
			enabled:       true,
			endpoint:      "non-existent",
			ttl:           time.Hour,
			testData:      nil,
			expectedFound: false,
		},
		{
			name:          "expired cache",
			enabled:       true,
			endpoint:      "test-endpoint",
			ttl:           10 * time.Millisecond,
			sleep:         20 * time.Millisecond,
			testData:      map[string]any{"test": "value"},
			expectedFound: false,
		},
		{
			name:          "disabled cache regular endpoint",
			enabled:       false,
			endpoint:      "test-endpoint",
			ttl:           time.Hour,
			testData:      map[string]any{"test": "value"},
			expectedFound: false,
		},
		{
			name:          "disabled cache full endpoint",
			enabled:       false,
			endpoint:      "products/full",
			ttl:           time.Hour,
			testData:      map[string]any{"test": "full"},
			expectedFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewCacheManager(t.TempDir(), DefaultBaseURL, tt.enabled, tt.ttl)

			if tt.testData != nil {
				err := cm.Set(tt.endpoint, tt.testData)
				if tt.expectSetError && err == nil {
					t.Error("Expected set error but got none")
				} else if !tt.expectSetError && err != nil {
					t.Fatalf("Failed to set cache: %v", err)
				}
			}

			if tt.sleep > 0 {
				time.Sleep(tt.sleep)
			}

			cached, found := cm.Get(tt.endpoint)
			if found != tt.expectedFound {
				t.Errorf("Expected found=%v, got %v", tt.expectedFound, found)
				return
			}

			if found && tt.testData != nil { //nolint:nestif // ok
				var cachedMap map[string]any
				if err := json.Unmarshal(cached, &cachedMap); err != nil {
					t.Fatalf("Failed to unmarshal cached data: %v", err)
				}

				for key, expected := range tt.testData {
					if key == "num" {
						if cachedMap[key] != float64(expected.(int)) {
							t.Errorf("Expected %s=%v, got %v", key, float64(expected.(int)), cachedMap[key])
						}
					} else if cachedMap[key] != expected {
						t.Errorf("Expected %s=%v, got %v", key, expected, cachedMap[key])
					}
				}
			}
		})
	}
}

//nolint:paralleltest,tparallel // t.TempDir
func TestCacheManagerClear(t *testing.T) {
	//nolint:govet // ok
	tests := []struct {
		name        string
		baseDir     string
		expectError bool
		errorType   error
		setupData   bool
	}{
		{
			name:      "successful clear allowed path",
			baseDir:   "eol-cache",
			setupData: true,
		},
		{
			name:      "successful clear dot eol cache",
			baseDir:   ".eol-cache",
			setupData: true,
		},
		{
			name:      "successful clear eol path",
			baseDir:   "eol",
			setupData: true,
		},
		{
			name:        "refuse dangerous path fake root",
			baseDir:     "/fake_root_that_does_not_exist",
			expectError: true,
			errorType:   errRefusingToClear,
		},
		{
			name:        "refuse dangerous path fake home",
			baseDir:     "/home/fake_user_that_does_not_exist",
			expectError: true,
			errorType:   errRefusingToClear,
		},
		{
			name:        "refuse dangerous path system dir",
			baseDir:     "/usr/fake_system_dir",
			expectError: true,
			errorType:   errRefusingToClear,
		},
		{
			name:        "refuse random temp dir",
			expectError: true,
			errorType:   errRefusingToClear,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			baseDir := tt.baseDir
			if tt.name == "refuse random temp dir" {
				baseDir = t.TempDir()
			} else if tt.baseDir != "" && !strings.HasPrefix(tt.baseDir, "/") {
				baseDir = filepath.Join(t.TempDir(), tt.baseDir)
			}

			cm := NewCacheManager(baseDir, DefaultBaseURL, true, time.Hour)

			if tt.setupData {
				testData := map[string]any{"test": "data"}
				if err := cm.Set("test-endpoint", testData); err != nil {
					t.Fatalf("Failed to set test data: %v", err)
				}

				_, found := cm.Get("test-endpoint")
				if !found {
					t.Fatal("Expected to find test data before clear")
				}
			}

			err := cm.Clear()
			if tt.expectError { //nolint:nestif // ok
				if err == nil {
					t.Errorf("Expected error when trying to clear path: %s", baseDir)
				} else if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error %v, got: %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for allowed path %s, got: %v", baseDir, err)
				}

				if tt.setupData {
					if _, found := cm.Get("test-endpoint"); found {
						t.Error("Expected cache to be cleared")
					}
				}
			}
		})
	}
}

//nolint:paralleltest // t.TempDir
func TestCacheManagerClearExpired(t *testing.T) {
	//nolint:govet // ok
	tests := []struct {
		name        string
		enabled     bool
		setupFunc   func(*testing.T, *CacheManager) error
		expectError bool
	}{
		{
			name:    "empty cache directory",
			enabled: true,
			setupFunc: func(t *testing.T, cm *CacheManager) error {
				t.Helper()
				return nil
			},
		},
		{
			name:    "disabled cache",
			enabled: false,
			setupFunc: func(t *testing.T, cm *CacheManager) error {
				t.Helper()
				return nil
			},
		},
		{
			name:    "valid and expired entries",
			enabled: true,
			setupFunc: func(t *testing.T, cm *CacheManager) error {
				t.Helper()

				validData := map[string]any{"valid": "data"}
				if err := cm.Set("valid-endpoint", validData); err != nil {
					return err
				}

				expiredEntry := CacheEntry{
					Timestamp: time.Now().Add(-2 * time.Hour),
					ExpiresAt: time.Now().Add(-time.Hour),
					Data:      json.RawMessage(`{"expired": true}`),
					Endpoint:  "expired-endpoint",
				}

				expiredJSON, err := json.MarshalIndent(expiredEntry, "", "  ")
				if err != nil {
					return err
				}

				expiredFile := filepath.Join(cm.baseDir, "expired"+cacheExt)

				return os.WriteFile(expiredFile, expiredJSON, 0o644)
			},
		},
		{
			name:    "cache with negative TTL entries",
			enabled: true,
			setupFunc: func(t *testing.T, cm *CacheManager) error {
				t.Helper()

				negativeTTLCM := NewCacheManager(cm.baseDir, DefaultBaseURL, true, -time.Hour)

				testData := map[string]any{"test": "data"}
				if err := negativeTTLCM.Set("test1", testData); err != nil {
					return err
				}

				return negativeTTLCM.Set("test2", testData)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := filepath.Join(t.TempDir(), "eol-cache")
			cm := NewCacheManager(tempDir, DefaultBaseURL, tt.enabled, time.Hour)

			if err := tt.setupFunc(t, cm); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			err := cm.ClearExpired()
			if tt.expectError { //nolint:nestif // ok
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ClearExpired failed: %v", err)
				}

				if tt.name == "valid and expired entries" {
					_, found := cm.Get("valid-endpoint")
					if !found {
						t.Error("Expected valid cache to still exist")
					}

					expiredFile := filepath.Join(tempDir, "expired"+cacheExt)
					if _, err = os.Stat(expiredFile); !os.IsNotExist(err) {
						t.Error("Expected expired cache file to be removed")
					}
				}
			}
		})
	}
}

//nolint:paralleltest // t.TempDir
func TestCacheManagerGetStats(t *testing.T) {
	//nolint:govet // ok
	tests := []struct {
		name         string
		enabled      bool
		setupFunc    func(*testing.T, *CacheManager)
		validateFunc func(*testing.T, CacheStats)
	}{
		{
			name:      "disabled cache",
			enabled:   false,
			setupFunc: func(t *testing.T, cm *CacheManager) { t.Helper() },
			validateFunc: func(t *testing.T, stats CacheStats) {
				t.Helper()

				if stats.Enabled != false {
					t.Errorf("Expected enabled=false, got %v", stats.Enabled)
				}

				if stats.TotalFiles != 0 {
					t.Errorf("Expected 0 files for disabled cache, got %d", stats.TotalFiles)
				}
			},
		},
		{
			name:      "empty cache directory",
			enabled:   true,
			setupFunc: func(t *testing.T, cm *CacheManager) { t.Helper() },
			validateFunc: func(t *testing.T, stats CacheStats) {
				t.Helper()

				if stats.Enabled != true {
					t.Errorf("Expected enabled=true, got %v", stats.Enabled)
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
			name:    "cache with files",
			enabled: true,
			setupFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				testData := map[string]any{"test": "value"}
				if err := cm.Set("test-endpoint", testData); err != nil {
					t.Fatalf("Failed to set cache: %v", err)
				}
			},
			validateFunc: func(t *testing.T, stats CacheStats) {
				t.Helper()

				if stats.Enabled != true {
					t.Errorf("Expected enabled=true, got %v", stats.Enabled)
				}

				if stats.TotalFiles < 1 {
					t.Errorf("Expected at least 1 total file, got %v", stats.TotalFiles)
				}

				if stats.ValidFiles < 1 {
					t.Errorf("Expected at least 1 valid file, got %v", stats.ValidFiles)
				}

				if stats.TotalSize == 0 {
					t.Error("Expected non-zero size for cache with files")
				}
			},
		},
		{
			name:    "cache with multiple files",
			enabled: true,
			setupFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				testData1 := map[string]any{"test": "data1"}
				testData2 := map[string]any{"test": "data2"}

				if err := cm.Set("test1", testData1); err != nil {
					t.Fatalf("Failed to set cache 1: %v", err)
				}

				if err := cm.Set("test2", testData2); err != nil {
					t.Fatalf("Failed to set cache 2: %v", err)
				}
			},
			validateFunc: func(t *testing.T, stats CacheStats) {
				t.Helper()

				if stats.TotalFiles < 2 {
					t.Errorf("Expected at least 2 files, got %d", stats.TotalFiles)
				}

				if stats.ValidFiles < 2 {
					t.Errorf("Expected at least 2 valid files, got %d", stats.ValidFiles)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			cm := NewCacheManager(tempDir, DefaultBaseURL, tt.enabled, time.Hour)

			tt.setupFunc(t, cm)

			stats, err := cm.GetStats()
			if err != nil {
				t.Fatalf("Failed to get stats: %v", err)
			}

			if stats.CacheDir != tempDir {
				t.Errorf("Expected cache_dir=%s, got %v", tempDir, stats.CacheDir)
			}

			tt.validateFunc(t, stats)
		})
	}
}

//nolint:paralleltest // t.TempDir
func TestCacheManagerSmartGet(t *testing.T) {
	fullProductsData := map[string]any{
		"schema_version": "1.2.0",
		"last_modified":  "2025-01-11T00:00:00Z",
		"result": []any{
			map[string]any{
				"name":     "go",
				"label":    "Go",
				"category": "lang",
				"tags":     []any{"language", "programming"},
				"releases": []any{
					map[string]any{
						"name":         "1.24",
						"label":        "1.24",
						"releaseDate":  "2025-02-11",
						"isLts":        false,
						"isMaintained": true,
						"isEol":        false,
					},
					map[string]any{
						"name":         "1.23",
						"label":        "1.23",
						"releaseDate":  "2024-08-13",
						"isLts":        false,
						"isMaintained": true,
						"isEol":        false,
					},
				},
			},
			map[string]any{
				"name":     "python",
				"label":    "Python",
				"category": "lang",
				"tags":     []any{"language", "scripting"},
				"releases": []any{
					map[string]any{
						"name":         "3.12",
						"label":        "3.12",
						"releaseDate":  "2023-10-02",
						"isLts":        false,
						"isMaintained": true,
						"isEol":        false,
					},
				},
			},
		},
	}
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
				},
			},
		},
	}
	tests := []struct { //nolint:govet // ok
		name            string
		endpoint        string
		params          []string
		setupFullCache  bool
		setupProdCache  bool
		setupExactCache bool
		expectFound     bool
		expectExtracted bool
		description     string
	}{
		{
			name:            "products from exact cache",
			endpoint:        "/products",
			setupExactCache: true,
			expectFound:     true,
			expectExtracted: false,
			description:     "Should find exact /products cache",
		},
		{
			name:            "products from full cache",
			endpoint:        "/products",
			setupFullCache:  true,
			expectFound:     true,
			expectExtracted: true,
			description:     "Should extract products list from /products/full cache",
		},
		{
			name:            "product from exact cache",
			endpoint:        "/products/go",
			params:          []string{"go"},
			setupExactCache: true,
			expectFound:     true,
			expectExtracted: false,
			description:     "Should find exact /products/go cache",
		},
		{
			name:            "product from full cache",
			endpoint:        "/products/go",
			params:          []string{"go"},
			setupFullCache:  true,
			expectFound:     true,
			expectExtracted: true,
			description:     "Should extract product from /products/full cache",
		},
		{
			name:            "release from exact cache",
			endpoint:        "/products/go/releases/1.24",
			params:          []string{"go", "1.24"},
			setupExactCache: true,
			expectFound:     true,
			expectExtracted: false,
			description:     "Should find exact release cache",
		},
		{
			name:            "release from product cache",
			endpoint:        "/products/go/releases/1.24",
			params:          []string{"go", "1.24"},
			setupProdCache:  true,
			expectFound:     true,
			expectExtracted: true,
			description:     "Should extract release from product cache",
		},
		{
			name:            "release from full cache",
			endpoint:        "/products/go/releases/1.24",
			params:          []string{"go", "1.24"},
			setupFullCache:  true,
			expectFound:     true,
			expectExtracted: true,
			description:     "Should extract release from full cache",
		},
		{
			name:            "categories from full cache",
			endpoint:        "/categories",
			setupFullCache:  true,
			expectFound:     true,
			expectExtracted: true,
			description:     "Should extract categories from full cache",
		},
		{
			name:            "products by category from full cache",
			endpoint:        "/categories/lang",
			params:          []string{"category", "lang"},
			setupFullCache:  true,
			expectFound:     true,
			expectExtracted: true,
			description:     "Should extract products by category from full cache",
		},
		{
			name:            "tags from full cache",
			endpoint:        "/tags",
			setupFullCache:  true,
			expectFound:     true,
			expectExtracted: true,
			description:     "Should extract tags from full cache",
		},
		{
			name:            "products by tag from full cache",
			endpoint:        "/tags/language",
			params:          []string{"tag", "language"},
			setupFullCache:  true,
			expectFound:     true,
			expectExtracted: true,
			description:     "Should extract products by tag from full cache",
		},
		{
			name:        "no cache available",
			endpoint:    "/products",
			expectFound: false,
			description: "Should return cache miss when no cache available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewCacheManager(t.TempDir(), DefaultBaseURL, true, time.Hour)

			if tt.setupFullCache {
				if err := cm.Set("/products/full", fullProductsData); err != nil {
					t.Fatalf("Failed to setup full cache: %v", err)
				}
			}

			if tt.setupProdCache {
				if err := cm.Set("/products/go", productData, "go"); err != nil {
					t.Fatalf("Failed to setup product cache: %v", err)
				}
			}

			if tt.setupExactCache {
				if err := cm.Set(tt.endpoint, map[string]any{"exact": "cache"}, tt.params...); err != nil {
					t.Fatalf("Failed to setup exact cache: %v", err)
				}
			}

			data, found := cm.Get(tt.endpoint, tt.params...)
			if found != tt.expectFound {
				t.Errorf("%s: Expected found=%v, got found=%v", tt.description, tt.expectFound, found)
				return
			}

			if found { //nolint:nestif // ok
				result := map[string]any{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Errorf("%s: Failed to unmarshal result: %v", tt.description, err)
				}

				if tt.expectExtracted {
					if _, hasSchema := result["schema_version"]; !hasSchema {
						t.Errorf("%s: Expected extracted data to have schema_version", tt.description)
					}

					if _, hasResult := result["result"]; !hasResult {
						t.Errorf("%s: Expected extracted data to have result", tt.description)
					}
				}
			}
		})
	}
}

func TestCacheManagerDynamicBaseURL(t *testing.T) {
	t.Parallel()

	customBaseURL := "https://custom.api.example.com/v2"
	cm := NewCacheManager(t.TempDir(), customBaseURL, true, time.Hour)
	fullProductsData := map[string]any{
		"schema_version": "1.2.0",
		"total":          2,
		"result": []any{
			map[string]any{
				"name":     "go",
				"label":    "Go",
				"category": "lang",
				"aliases":  []any{"golang"},
				"tags":     []any{"google", "programming"},
			},
			map[string]any{
				"name":     "ubuntu",
				"label":    "Ubuntu",
				"category": "os",
				"tags":     []any{"canonical", "linux"},
			},
		},
	}

	err := cm.Set("/products/full", fullProductsData)
	if err != nil {
		t.Fatalf("Failed to set full products cache: %v", err)
	}

	productsData, found := cm.Get("/products")
	if !found {
		t.Fatal("Expected to find products data extracted from full cache")
	}

	productsResponse := map[string]any{}
	if err = json.Unmarshal(productsData, &productsResponse); err != nil {
		t.Fatalf("Failed to unmarshal products response: %v", err)
	}

	result, ok := productsResponse["result"].([]any)
	if !ok {
		t.Fatal("Expected result to be array")
	}

	for _, item := range result {
		product, ok2 := item.(map[string]any)
		if !ok2 {
			continue
		}

		uri, ok2 := product["uri"].(string)
		if !ok2 {
			t.Errorf("Expected uri field in product %v", product["name"])
			continue
		}

		expectedPrefix := customBaseURL + "/products/"
		if !strings.HasPrefix(uri, expectedPrefix) {
			t.Errorf("Expected URI to start with %s, got %s", expectedPrefix, uri)
		}
	}

	categoriesData, found := cm.Get("/categories")
	if !found {
		t.Fatal("Expected to find categories data extracted from full cache")
	}

	catResp := map[string]any{}
	if err = json.Unmarshal(categoriesData, &catResp); err != nil {
		t.Fatalf("Failed to unmarshal categories response: %v", err)
	}

	categoryResult, ok := catResp["result"].([]any)
	if !ok {
		t.Fatal("Expected categories result to be array")
	}

	for _, item := range categoryResult {
		category, ok2 := item.(map[string]any)
		if !ok2 {
			continue
		}

		uri, ok2 := category["uri"].(string)
		if !ok2 {
			t.Errorf("Expected uri field in category %v", category["name"])
			continue
		}

		expectedPrefix := customBaseURL + "/categories/"
		if !strings.HasPrefix(uri, expectedPrefix) {
			t.Errorf("Expected category URI to start with %s, got %s", expectedPrefix, uri)
		}
	}
}
