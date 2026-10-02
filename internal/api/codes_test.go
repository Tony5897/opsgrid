package api_test

import "github.com/Tony5897/opsgrid/internal/platform/apperr"

func apperrCodes() []string {
	var out []string
	for _, c := range apperr.AllCodes() {
		out = append(out, string(c))
	}
	return out
}
