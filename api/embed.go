// Package apispec embeds the OpenAPI contract so tests can validate real
// responses against it.
package apispec

import _ "embed"

// OpenAPI is the raw contract (api/openapi.yaml).
//
//go:embed openapi.yaml
var OpenAPI []byte
