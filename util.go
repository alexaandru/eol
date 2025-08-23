package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// generateVersionVariants generates all possible version variants for a given version string
// by progressively removing segments from the end, separated by dots.
// For example: "1.2.3.4" -> ["1.2.3.4", "1.2.3", "1.2", "1"]
// Returns them in order of specificity: most specific to least specific.
func generateVersionVariants(version string) (variants []string) {
	ver := strings.TrimSpace(version)
	if ver == "" {
		return
	}

	current := ver
	variants = append(variants, current)

	for {
		lastDot := strings.LastIndex(current, ".")
		if lastDot == -1 {
			return
		}

		current = current[:lastDot]
		if current != "" {
			variants = append(variants, current)
		}
	}
}

func parseExtendedDuration(dur string) (time.Duration, error) {
	if dur = strings.TrimSpace(dur); dur == "" {
		return 0, fmt.Errorf("%w: %q", errInvalidDuration, dur)
	}

	matches := reCustomDur.FindStringSubmatch(dur)
	if matches == nil {
		return time.ParseDuration(dur) //nolint:wrapcheck // ok
	}

	num, _ := strconv.Atoi(matches[1]) //nolint:errcheck // we used a regex to validate
	unit, hours := matches[2], 0

	//nolint:mnd // ok
	switch unit {
	case "d":
		hours = num * 24
	case "wk":
		hours = num * 7 * 24
	case "mo":
		hours = num * 30 * 24
	}

	return time.ParseDuration(fmt.Sprintf("%dh", hours)) //nolint:wrapcheck // ok
}

func buildURL(u url.URL, endpoint string) string { //nolint:gocritic // ok
	u.Path = path.Join(u.Path, endpoint)
	return u.String()
}

func toJSON(v any) string {
	b, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return string(b)
}

func eolWithin(duration string, eolDate any) (ok bool) {
	var err error

	defer func() {
		if err != nil {
			panic(fmt.Errorf("%w: %w", errInvalidDuration, err))
		}
	}()

	dur, err := parseExtendedDuration(duration)
	if err != nil {
		return
	}

	if eolDate == nil {
		return
	}

	var dateStr string

	switch v := eolDate.(type) {
	case *string:
		if v == nil {
			return
		}

		dateStr = *v
	case string:
		dateStr = v
	default:
		return
	}

	if dateStr == "" {
		return
	}

	eolTime, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return
	}

	now := time.Now()
	futureLimit := now.Add(dur)

	return eolTime.After(now) && eolTime.Before(futureLimit)
}

func dict(values ...any) (dict map[string]any, err error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("%w: requires an even number of arguments", errInvalidDict)
	}

	dict = make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("%w: keys must be strings", errInvalidDict)
		}

		dict[key] = values[i+1]
	}

	return
}

func toStringSlice(slice any) (result []string) {
	if slice == nil {
		return
	}

	switch v := slice.(type) {
	case []any:
		result = make([]string, len(v))
		for i, item := range v {
			if str, ok := item.(string); ok {
				result[i] = str
			} else {
				result[i] = fmt.Sprintf("%v", item)
			}
		}

		return
	case []string:
		return v
	default:
		return []string{fmt.Sprintf("%v", slice)}
	}
}

func collect(field string, slice any) (result []any) {
	if slice == nil {
		return
	}

	switch v := slice.(type) {
	case []any:
		for _, item := range v {
			if itemMap, ok := item.(map[string]any); ok {
				if value, exists := itemMap[field]; exists {
					result = append(result, value)
				}
			}
		}

		return
	default:
		return
	}
}

func configDir(opts ...string) string {
	var xs []string

	switch homeDir, err := os.UserHomeDir(); {
	case err != nil:
		xs = []string{".eol"}
	case runtime.GOOS == "windows":
		xs = []string{homeDir, "AppData", "Local", "eol"}
	case runtime.GOOS == "darwin":
		xs = []string{homeDir, "Library", "Application Support", "eol"}
	default:
		xs = []string{homeDir, ".config", "eol"}
	}

	return filepath.Join(append(xs, opts...)...)
}
