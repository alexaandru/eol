package main

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"testing"
	"time"
)

func TestGenerateVersionVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		exp     []string
	}{
		{"", nil},
		{"  ", nil},
		{" \t\t\t\n ", nil},
		{"foo", []string{"foo"}},
		{"foo.bar", []string{"foo.bar", "foo"}},
		{"foo.bar.baz.foobar", []string{"foo.bar.baz.foobar", "foo.bar.baz", "foo.bar", "foo"}},
		{"1.2.3.4", []string{"1.2.3.4", "1.2.3", "1.2", "1"}},
		{"1.2.3", []string{"1.2.3", "1.2", "1"}},
		{"1.2", []string{"1.2", "1"}},
		{"1", []string{"1"}},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()

			got := generateVersionVariants(tc.version)
			if !slices.Equal(got, tc.exp) {
				t.Fatalf("expected %q, got %q", tc.exp, got)
			}
		})
	}
}

func TestParseExtendedDuration(t *testing.T) {
	t.Parallel()

	//nolint:govet // ok
	tests := []struct {
		dur    string
		exp    time.Duration
		expErr error
	}{
		{"", 0, errInvalidDuration},
		{"1h1m1s", 3661000000000, nil},
		{"10d", 864000000000000, nil},
		{"4wk", 2419200000000000, nil},
		{"2mo", 5184000000000000, nil},
	}

	for _, tc := range tests {
		t.Run(tc.dur, func(t *testing.T) {
			t.Parallel()

			got, err := parseExtendedDuration(tc.dur)
			if !errors.Is(err, tc.expErr) {
				t.Fatalf("expected error %v, got %v", tc.expErr, err)
			}

			if got != tc.exp {
				t.Fatalf("expected duration %v, got %v", tc.exp, got)
			}
		})
	}
}

func TestBuildURL(t *testing.T) {
	t.Parallel()

	u1, _ := url.Parse("https://example.com/api")
	u2, _ := url.Parse("https://example.com/api/")

	tests := []struct {
		baseURL       *url.URL
		endpoint, exp string
	}{
		{u1, "/products/123", "https://example.com/api/products/123"},
		{u2, "/products/123", "https://example.com/api/products/123"},
		{u1, "products/123", "https://example.com/api/products/123"},
		{u2, "products/123", "https://example.com/api/products/123"},
		{u2, "/", "https://example.com/api"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s--%s", tc.baseURL, tc.endpoint), func(t *testing.T) {
			t.Parallel()

			got := buildURL(*tc.baseURL, tc.endpoint)
			if got != tc.exp {
				t.Fatalf("expected URL %q, got %q", tc.exp, got)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v   any
		exp string
	}{
		{map[string]any{"foo": "bar"}, "{\n    \"foo\": \"bar\"\n  }"},
		{[]int{1, 2, 3}, "[\n    1,\n    2,\n    3\n  ]"},
		{"hello", "\"hello\""},
		{123, "123"},
		{func() {}, "error: json: unsupported type: func()"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%v", tc.v), func(t *testing.T) {
			t.Parallel()

			got := toJSON(tc.v)
			if got != tc.exp {
				t.Fatalf("expected JSON %q, got %q", tc.exp, got)
			}
		})
	}
}

func TestEolWithin(t *testing.T) {
	t.Parallel()

	var z *string

	now := time.Now()
	//nolint:govet // ok
	tests := []struct {
		duration string
		eolDate  any
		exp      bool
		expErr   error
	}{
		{"10d", now.Add(5 * 24 * time.Hour).Format("2006-01-02"), true, nil},
		{"10d", now.Add(15 * 24 * time.Hour).Format("2006-01-02"), false, nil},
		{"4wk", now.Add(2 * 7 * 24 * time.Hour).Format("2006-01-02"), true, nil},
		{"4wk", now.Add(5 * 7 * 24 * time.Hour).Format("2006-01-02"), false, nil},
		{"2mo", now.Add(30 * 24 * time.Hour).Format("2006-01-02"), true, nil},
		{"2mo", now.Add(70 * 24 * time.Hour).Format("2006-01-02"), false, nil},
		{"10d", nil, false, nil},
		{"10d", p(now.Add(5 * 24 * time.Hour).Format("2006-01-02")), true, nil},
		{"10d", p(""), false, nil},
		{"10d", 123, false, nil},
		{"10d", z, false, nil},
		{"", nil, false, errInvalidDuration},
		{"10d", "invalid-date", false, errInvalidDuration},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s--%v", tc.duration, tc.eolDate), func(t *testing.T) {
			t.Parallel()

			var (
				got bool
				err error
			)

			func() {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic: %w", r.(error))
					}
				}()

				got = eolWithin(tc.duration, tc.eolDate)
			}()

			if !errors.Is(err, tc.expErr) {
				t.Fatalf("expected error %v, got %v", tc.expErr, err)
			} else if got != tc.exp {
				t.Fatalf("expected %v, got %v", tc.exp, got)
			}
		})
	}
}

func TestDict(t *testing.T) {
	t.Parallel()

	//nolint:govet // ok
	tests := []struct {
		values []any
		exp    map[string]any
		expErr error
	}{
		{[]any{"key1", "value1", "key2", 42}, map[string]any{"key1": "value1", "key2": 42}, nil},
		{[]any{"key1", "value1", "key2"}, nil, errInvalidDict},
		{[]any{1, "value1", 2, "value2"}, nil, errInvalidDict},
		{[]any{}, map[string]any{}, nil},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%v", tc.values), func(t *testing.T) {
			t.Parallel()

			got, err := dict(tc.values...)
			if !errors.Is(err, tc.expErr) {
				t.Fatalf("expected error %v, got %v", tc.expErr, err)
			}

			if !maps.Equal(got, tc.exp) {
				t.Fatalf("expected map %v, got %v", tc.exp, got)
			}
		})
	}
}

func TestToStringSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v   any
		exp []string
	}{
		{[]string{"foo", "bar"}, []string{"foo", "bar"}},
		{[]any{"foo", "bar"}, []string{"foo", "bar"}},
		{[]any{"foo", 42}, []string{"foo", "42"}},
		{[]any(nil), nil},
		{nil, nil},
		{"not a slice", []string{"not a slice"}},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%v", tc.v), func(t *testing.T) {
			t.Parallel()

			got := toStringSlice(tc.v)
			if !slices.Equal(got, tc.exp) {
				t.Fatalf("expected slice %v, got %#v", tc.exp, got)
			}
		})
	}
}

func TestCollect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		field string
		v     any
		exp   []any
	}{
		{"name", []any{map[string]any{"name": "Alice"}, map[string]any{"name": "Bob"}}, []any{"Alice", "Bob"}},
		{"age", []any{map[string]any{"name": "Alice"}, map[string]any{"age": 30}}, []any{30}},
		{"name", []any{map[string]any{"age": 25}}, nil},
		{"name", nil, nil},
		{"name", "not a slice", nil},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s--%v", tc.field, tc.v), func(t *testing.T) {
			t.Parallel()

			got := collect(tc.field, tc.v)
			if !slices.Equal(got, tc.exp) {
				t.Fatalf("expected collected values %v, got %v", tc.exp, got)
			}
		})
	}
}

func TestConfigDir(t *testing.T) {
	t.Parallel()

	dir := configDir("eoltest")
	if dir == "" {
		t.Fatal("expected non-empty directory path")
	}
}

func p[T any](v T) *T {
	return &v
}
