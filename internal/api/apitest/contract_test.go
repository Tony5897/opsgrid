package apitest_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tony5897/opsgrid/internal/api/apitest"
)

// The harness must reject drift, or a green API suite proves nothing.
func TestCheckRejectsDrift(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		path    string
		handler http.HandlerFunc
		want    string
	}{
		{
			name: "wrong const value",
			path: "/v1/meta",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"service":"not-opsgrid","version":"dev","commit":""}`)
			},
			want: "response 200 violates contract",
		},
		{
			name: "undocumented property (additionalProperties: false)",
			path: "/v1/meta",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"service":"opsgrid","version":"dev","commit":"","secret":"x"}`)
			},
			want: "response 200 violates contract",
		},
		{
			name: "problem without a code",
			path: "/v1/meta",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, `{"type":"about:blank","title":"x","status":500}`)
			},
			want: "response 500 violates contract",
		},
		{
			name:    "undocumented route",
			path:    "/v1/not-in-contract",
			handler: func(http.ResponseWriter, *http.Request) {},
			want:    "is not in the contract",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := apitest.Check(tc.handler, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}
