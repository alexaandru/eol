package eol

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"
)

// Client represents an endoflife.date API client.
type Client struct {
	sink            io.Writer
	response        any
	baseURL         *url.URL
	httpClient      *http.Client
	cacheManager    *CacheManager
	config          *Config
	templateManager *TemplateManager
	userAgent       string
	responseHeader  string
	initialArgs     []string
}

// Option represents a functional option for configuring a Client.
type Option func(*Client)

// Default values.
const (
	DefaultTimeout  = 30 * time.Second
	DefaultCacheTTL = time.Hour
	UserAgent       = "eol-go-client/1.0"
	DefaultBaseURL  = "https://endoflife.date/api/v1"
)

// New creates a new endoflife.date API client with default settings.
//
//nolint:gocognit // ok
func New(opts ...Option) (c *Client, err error) {
	c = &Client{userAgent: UserAgent}

	for _, opt := range opts {
		opt(c)
	}

	if c.baseURL == nil {
		c.baseURL, err = url.Parse(DefaultBaseURL)
		if err != nil {
			return
		}
	}

	if c.initialArgs == nil {
		c.initialArgs = os.Args[1:]
	}

	if c.config == nil {
		c.config, err = NewConfig(c.initialArgs...)
		if err != nil {
			return
		}
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: DefaultTimeout}
	}

	if c.cacheManager == nil {
		c.cacheManager = NewCacheManager(c.config.CacheDir, c.baseURL.String(), c.config.CacheEnabled,
			cmp.Or(c.config.CacheTTL, DefaultCacheTTL))
	}

	if c.templateManager == nil { //nolint:nestif // ok
		templateDir := c.config.TemplateDir
		if templateDir == "" {
			var homeDir string

			homeDir, err = os.UserHomeDir()
			if err != nil {
				return
			}

			defaultTemplateDir := filepath.Join(homeDir, ".config", "eol", "templates")
			if _, statErr := os.Stat(defaultTemplateDir); statErr == nil {
				templateDir = defaultTemplateDir
			}
		}

		c.templateManager, err = NewTemplateManager(templateDir,
			c.config.InlineTemplate, c.config.Command, c.config.Args)
		if err != nil {
			return
		}
	}

	if c.sink == nil {
		c.sink = os.Stdout
	}

	return
}

// WithInitialArgs returns an Option that sets the initial command-line arguments for the client.
func WithInitialArgs(args []string) Option {
	return func(c *Client) {
		c.initialArgs = args
	}
}

// WithBaseURL returns an Option that sets the base URL for API requests.
func WithBaseURL(baseURL *url.URL) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithConfig returns an Option that sets the configuration for the client.
func WithConfig(cfg *Config) Option {
	return func(c *Client) {
		c.config = cfg
	}
}

// WithHTTPClient returns an Option that sets the HTTP client for making requests.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithCacheManager returns an Option that sets the cache manager for the client.
func WithCacheManager(cm *CacheManager) Option {
	return func(c *Client) {
		c.cacheManager = cm
	}
}

// WithTemplateManager returns an Option that sets the template manager for the client.
func WithTemplateManager(tm *TemplateManager) Option {
	return func(c *Client) {
		c.templateManager = tm
	}
}

// WithSink returns an Option that sets the output writer for the client.
func WithSink(sink io.Writer) Option {
	return func(c *Client) {
		c.sink = sink
	}
}

// buildURL constructs a URL for the given endpoint path.
func (c *Client) buildURL(endpoint string) string {
	u := *c.baseURL
	if endpoint == "/" {
		u.Path += "/"
	} else {
		u.Path = path.Join(u.Path, endpoint)
	}

	return u.String()
}

// doRequestWithCache performs an HTTP GET request, with caching support.
func (c *Client) doRequest(endpoint string, result any, params ...string) (err error) {
	if cached, found := c.cacheManager.Get(endpoint, params...); found {
		if err = json.Unmarshal(cached, result); err == nil {
			return // Cache hit.
		}
	}

	urL := c.buildURL(endpoint)

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, urL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // ok

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s (%d)", http.StatusText(resp.StatusCode), resp.StatusCode) //nolint:err113 // ok
	}

	if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return c.cacheManager.Set(endpoint, result, params...)
}
