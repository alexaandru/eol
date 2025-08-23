// Package templates exports embedded templates for use in the application.
package templates

import "embed"

//go:embed *.tmpl
var Templates embed.FS //nolint:revive // ok
