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

func TestCacheManagerGenerateCacheKey(t *testing.T) {
	t.Parallel()

	cm := NewCacheManager("/tmp/test", true, time.Hour)

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

	cm := NewCacheManager("/tmp/test", true, time.Hour)

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

	cm := NewCacheManager("/tmp/test", true, time.Hour)

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
			cm := NewCacheManager(t.TempDir(), tt.enabled, tt.ttl)

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

			cm := NewCacheManager(baseDir, true, time.Hour)

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

				negativeTTLCM := NewCacheManager(cm.baseDir, true, -time.Hour)

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
			cm := NewCacheManager(tempDir, tt.enabled, time.Hour)

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
			cm := NewCacheManager(tempDir, tt.enabled, time.Hour)

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

//nolint:paralleltest,tparallel // t.TempDir
func TestCacheManagerGetReleaseFromProductCache(t *testing.T) {
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

	//nolint:govet // ok
	tests := []struct {
		name            string
		enabled         bool
		product         string
		release         string
		setupCache      bool
		expectedFound   bool
		expectedName    string
		expectedVersion string
	}{
		{
			name:            "existing release",
			enabled:         true,
			product:         "go",
			release:         "1.24",
			setupCache:      true,
			expectedFound:   true,
			expectedName:    "1.24",
			expectedVersion: "1.2.0",
		},
		{
			name:            "another existing release",
			enabled:         true,
			product:         "go",
			release:         "1.23",
			setupCache:      true,
			expectedFound:   true,
			expectedName:    "1.23",
			expectedVersion: "1.2.0",
		},
		{
			name:          "non-existent release",
			enabled:       true,
			product:       "go",
			release:       "1.22",
			setupCache:    true,
			expectedFound: false,
		},
		{
			name:          "non-existent product",
			enabled:       true,
			product:       "nonexistent",
			release:       "1.0",
			setupCache:    true,
			expectedFound: false,
		},
		{
			name:            "version normalization",
			enabled:         true,
			product:         "go",
			release:         "1.23.4",
			setupCache:      true,
			expectedFound:   true,
			expectedName:    "1.23",
			expectedVersion: "1.2.0",
		},
		{
			name:          "cache disabled",
			enabled:       false,
			product:       "go",
			release:       "1.24",
			setupCache:    true,
			expectedFound: false,
		},
		{
			name:          "no cache data",
			enabled:       true,
			product:       "go",
			release:       "1.24",
			setupCache:    false,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := NewCacheManager(t.TempDir(), tt.enabled, time.Hour)

			if tt.setupCache {
				if err := cm.Set("/products/go", productData, "go"); err != nil {
					t.Fatalf("Failed to cache product data: %v", err)
				}
			}

			releaseData, found := cm.GetReleaseFromProductCache(tt.product, tt.release)

			if found != tt.expectedFound {
				t.Errorf("Expected found=%v, got found=%v", tt.expectedFound, found)
				return
			}

			if !tt.expectedFound {
				return // Test passed - we expected not to find it.
			}

			releaseResponse := map[string]any{}
			if err := json.Unmarshal(releaseData, &releaseResponse); err != nil {
				t.Fatalf("Failed to unmarshal release response: %v", err)
			}

			if schema, ok := releaseResponse["schema_version"].(string); !ok || schema != tt.expectedVersion {
				t.Errorf("Expected schema_version=%s, got %v", tt.expectedVersion, releaseResponse["schema_version"])
			}

			result, ok := releaseResponse["result"].(map[string]any)
			if !ok {
				t.Fatalf("Expected result to be a map, got %T", releaseResponse["result"])
			}

			if name, ok2 := result["name"].(string); !ok2 || name != tt.expectedName {
				t.Errorf("Expected release name=%s, got %v", tt.expectedName, result["name"])
			}
		})
	}
}

//nolint:paralleltest,tparallel // t.TempDir
func TestCacheManagerGetProductFromFullCache(t *testing.T) {
	//nolint:govet // ok
	tests := []struct {
		name          string
		enabled       bool
		product       string
		expectedFound bool
	}{
		{
			name:          "no cached data enabled",
			enabled:       true,
			product:       "go",
			expectedFound: false,
		},
		{
			name:          "no cached data disabled",
			enabled:       false,
			product:       "go",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := NewCacheManager(t.TempDir(), tt.enabled, time.Hour)

			_, found := cm.GetProductFromFullCache(tt.product)
			if found != tt.expectedFound {
				t.Errorf("Expected found=%v, got found=%v", tt.expectedFound, found)
			}
		})
	}
}

//nolint:paralleltest,tparallel // t.TempDir
func TestCacheManagerGetReleaseFromFullCache(t *testing.T) {
	//nolint:govet // ok
	tests := []struct {
		name          string
		enabled       bool
		product       string
		release       string
		expectedFound bool
	}{
		{
			name:          "no cached data enabled",
			enabled:       true,
			product:       "go",
			release:       "1.24",
			expectedFound: false,
		},
		{
			name:          "no cached data disabled",
			enabled:       false,
			product:       "go",
			release:       "1.24",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := NewCacheManager(t.TempDir(), tt.enabled, time.Hour)

			_, found := cm.GetReleaseFromFullCache(tt.product, tt.release)
			if found != tt.expectedFound {
				t.Errorf("Expected found=%v, got found=%v", tt.expectedFound, found)
			}
		})
	}
}

//nolint:paralleltest,tparallel // t.TempDir
func TestCacheManagerGetProductsFromFullCache(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		expectedFound bool
	}{
		{
			name:          "no cached data enabled",
			enabled:       true,
			expectedFound: false,
		},
		{
			name:          "no cached data disabled",
			enabled:       false,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := NewCacheManager(t.TempDir(), tt.enabled, time.Hour)

			_, found := cm.GetProductsFromFullCache()
			if found != tt.expectedFound {
				t.Errorf("Expected found=%v, got found=%v", tt.expectedFound, found)
			}
		})
	}
}

//nolint:paralleltest,tparallel // t.TempDir
func TestCacheManagerInvalidJSON(t *testing.T) {
	//nolint:govet // ok
	tests := []struct {
		name     string
		testFunc func(*testing.T, *CacheManager)
	}{
		{
			name: "GetProductFromFullCache with invalid JSON",
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				_, found := cm.GetProductFromFullCache("test")
				if found {
					t.Error("Expected not to find product with invalid JSON")
				}
			},
		},
		{
			name: "GetReleaseFromFullCache with invalid JSON",
			testFunc: func(t *testing.T, cm *CacheManager) {
				t.Helper()

				_, found := cm.GetReleaseFromFullCache("test", "1.0")
				if found {
					t.Error("Expected not to find release with invalid JSON")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := NewCacheManager(t.TempDir(), true, time.Hour)

			invalidJSON := []byte("{invalid json")
			if err := cm.Set("/test", invalidJSON, "test"); err != nil {
				t.Fatalf("Failed to set invalid JSON: %v", err)
			}

			tt.testFunc(t, cm)
		})
	}
}
