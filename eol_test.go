package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type mockHTTPClient struct{}

func TestNew(t *testing.T) {
	t.Parallel()

	c, _ := newClient([]string{"release", "go", "1.24", "-f", "text", "-t", "{{.}}"})

	if c.sink != os.Stdout {
		t.Fatalf("Expected sink to be os.Stdout, got %v", c.sink)
	}

	if x := c.baseURL.String(); x != DefaultBaseURL {
		t.Fatalf("Expected baseURL to be %q, got %q", DefaultBaseURL, x)
	}

	if x := c.format; x != FormatText {
		t.Fatalf("Expected format to be 'text', got %q", x)
	}

	if x := c.inlineTemplate; x != "{{.}}" {
		t.Fatalf("Expected templateStr to be '{{.}}', got %q", x)
	}

	if x := c.command; x != "release" {
		t.Fatalf("Expected command to be 'release', got %q", x)
	}

	if x := c.args; !slices.Equal(x, []string{"go", "1.24"}) {
		t.Fatalf("Expected args to be [go 1.24], got %v", x)
	}

	if c.httpClient == nil {
		t.Fatal("Expected httpClient to be non-nil")
	}

	if c.templates == nil {
		t.Fatal("Expected templates to be non-nil")
	}
}

func TestClientHandle(t *testing.T) {
	t.Parallel()

	//nolint:govet // ok
	cases := []struct {
		args     string
		expError error
	}{
		{"-h", nil},
		{"--help", nil},
		{"help", nil},
		{"version", nil},
		{"index", nil},
		{"index -t '{{.}}'", nil},
		{"index -f json", nil},
		{"index -t json", errInlineTemplate},
		{"products", nil},
		{"products-full", nil},
		{"product go", nil},
		{"product nokia", nil},
		{"product aws-lambda", nil},
		{"release ubuntu 22.04", nil},
		{"release go 1.24.6.100", nil},
		{"release go 1.24.6", nil},
		{"release go 1.24", nil},
		{"release go 1", errReleaseNotFound},
		{"latest ubuntu", nil},
		{"categories", nil},
		{"category os", nil},
		{"tags", nil},
		{"tag lang", nil},
		{"identifiers", nil},
		{"identifier purl", nil},
		{"completion", nil},
		{"completion-bash", nil},
		{"completion-zsh", nil},
		{"templates-export --templates-dir testdata/export1", nil},
		{"bogus", errUsage},
	}

	for _, tc := range cases {
		t.Run(tc.args, func(t *testing.T) {
			t.Parallel()

			c, err := newClient(strings.Split(tc.args, " "))
			if err != nil && !errors.Is(err, tc.expError) {
				t.Fatalf("Unexpected error: %v", err)
			}

			if err != nil {
				return
			}

			buf := &bytes.Buffer{}
			c.sink = buf
			c.httpClient = &mockHTTPClient{}

			if err = c.handle(); !errors.Is(err, tc.expError) { //nolint:nestif // ok
				t.Fatalf("Expected error %v, got %v", tc.expError, err)
			} else if err == nil {
				args := append([]string{c.command}, c.args...)
				if c.format != FormatText {
					args = append(args, "json")
				}

				if c.inlineTemplate != "" {
					args = append(args, fmt.Sprintf("inline%d", len(c.inlineTemplate)))
				}

				if c.inlineTemplate != "" {
					args = append(args, "inline")
				} else if c.templatesDir != "" {
					args = append(args, filepath.Base(c.templatesDir))
				}

				fname := strings.ReplaceAll(strings.Join(args, "_"), " ", "_")
				fname = filepath.Join("testdata", "handle", fname)
				t.Logf("Handle golden copy: %q", fname)

				var exp []byte

				exp, err = os.ReadFile(fname)
				if err != nil {
					t.Fatalf("Failed to read expected output from %q: %v", fname, err)
				}

				t.Logf("Golden copy file: %q", fname)

				if x := buf.String(); x != string(exp) {
					t.Fatalf("Expected output to contain %q, got %q", exp, x)
				}
			}
		})
	}
}

func TestClientParseFlags(t *testing.T) {
	t.Parallel()

	//nolint:govet // ok
	cases := []struct {
		args   []string
		exp    *client
		expErr error
	}{
		{nil, nil, errUsage},
		{[]string{"-f"}, nil, errUsage},
		{[]string{"--formate"}, nil, errUsage},
		{[]string{"-t"}, nil, errUsage},
		{[]string{"--template"}, nil, errUsage},
		{[]string{"--templates-dir"}, nil, errUsage},
		{[]string{"-f", "json"}, nil, errUsage},
		{[]string{"--format", "json"}, nil, errUsage},
		{[]string{"-f", "text"}, nil, errUsage},
		{[]string{"--format", "text"}, nil, errUsage},
		{[]string{"-f", "xml"}, nil, errUnsupportedFormat},
		{[]string{"--format", "xml"}, nil, errUnsupportedFormat},
		{[]string{"-t", "json"}, nil, errInlineTemplate},
		{[]string{"--template", "json"}, nil, errInlineTemplate},
		{[]string{"--templates-dir", ".eol"}, nil, errUsage},
		{[]string{"release"}, &client{command: "release"}, errUsage},
		{[]string{"product"}, &client{command: "product"}, errUsage},
		{[]string{"category"}, &client{command: "category"}, errUsage},
		{[]string{"tag"}, &client{command: "tag"}, errUsage},
		{[]string{"identifier"}, &client{command: "identifier"}, errUsage},
		{[]string{"latest"}, &client{command: "latest"}, errUsage},
		{[]string{"completion"}, &client{command: "completion-bash"}, nil},
		{[]string{"categories"}, &client{command: "categories"}, nil},
		{[]string{"tags"}, &client{command: "tags"}, nil},
		{[]string{"identifiers"}, &client{command: "identifiers"}, nil},
		{[]string{"index"}, &client{command: "index"}, nil},
		{[]string{"index", "--templates-dir"}, &client{command: "index"}, errUsage},
		{[]string{"release", "go"}, &client{command: "release", args: []string{"go"}}, errUsage},
		{[]string{"release", "go", "1.24"}, &client{command: "release", args: []string{"go", "1.24"}}, nil},
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			c := &client{}
			if err := c.parseFlags(tc.args); !errors.Is(err, tc.expErr) {
				t.Fatalf("Expected error %v, got %v", tc.expErr, err)
			} else if err == nil && (tc.exp == nil || !reflect.DeepEqual(c, tc.exp)) {
				t.Fatalf("Expected client %+v, got %+v", tc.exp, c)
			}
		})
	}
}

func TestClientExecuteTemplate(t *testing.T) {
	t.Parallel()
	t.Skip("Tested indirectly in TestClientHandle")
}

func TestLoadTemplates(t *testing.T) {
	t.Parallel()
	t.Skip("Tested indirectly in TestNew")
}

func TestClientDoRequest(t *testing.T) {
	t.Parallel()
	t.Skip("Tested indirectly in TestClientHandle")
}

func (m *mockHTTPClient) Do(r *http.Request) (w *http.Response, err error) {
	fname := "index"
	if r.URL.Path != "/" {
		fname = strings.ReplaceAll(strings.TrimLeft(r.URL.Path, "/"), "/", "_")
	}

	fname = filepath.Join("testdata", "golden", fname)

	content, err := os.ReadFile(fname)
	if err == nil {
		code := http.StatusOK
		if bytes.Contains(content, []byte("Page not Found")) {
			code = http.StatusNotFound
			err = errNotFound
		}

		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, err
	}

	w, err = http.DefaultClient.Do(r)
	if err == nil {
		body, _ := io.ReadAll(w.Body)
		os.WriteFile(fname, body, os.ModePerm)
		w.Body = io.NopCloser(bytes.NewReader(body))
	}

	return
}
