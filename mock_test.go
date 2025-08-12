package eol

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type mockResponse struct {
	Body string
	Code int
}

type mockTransport struct {
	responses map[string]*mockResponse
	err       error
}

func newMockClient(responses map[string]*mockResponse) *http.Client {
	return &http.Client{
		Transport: &mockTransport{
			responses: responses,
			err:       nil,
		},
	}
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	if resp, ok := m.responses[req.URL.String()]; ok {
		return newMockResponse(resp.Code, resp.Body), nil
	}

	// Default 404 response.
	return newMockResponse(http.StatusNotFound, "Not Found"), nil
}

func newMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}
}

func newClientWithTempCache(t *testing.T, httpClient *http.Client) (c *Client) {
	t.Helper()

	var (
		err    error
		config = &Config{TemplateDir: t.TempDir()}
	)

	c, err = New(
		WithHTTPClient(httpClient),
		WithCacheManager(NewCacheManager(t.TempDir(), DefaultBaseURL, true, time.Hour)),
		WithConfig(config),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return
}
