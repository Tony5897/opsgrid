package api_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tony5897/opsgrid/internal/api"
	"github.com/Tony5897/opsgrid/internal/api/apitest"
	"github.com/Tony5897/opsgrid/internal/api/gen"
)

func handler() http.Handler {
	return api.Handler(&api.Server{}, http.NewServeMux(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestGetMetaConformsToContract(t *testing.T) {
	t.Parallel()
	rr := apitest.Do(t, handler(), httptest.NewRequest(http.MethodGet, "/v1/meta", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var m gen.Meta
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m.Service != gen.Opsgrid || m.Version == "" {
		t.Fatalf("unexpected meta: %+v", m)
	}
}

// Every stable error code in the Go error model must exist in the contract's
// ProblemCode enum, and vice versa, so clients can switch on codes safely.
func TestProblemCodesMatchContract(t *testing.T) {
	t.Parallel()
	contract := map[string]bool{}
	for _, c := range apitest.SchemaEnum(t, "ProblemCode") {
		contract[c] = true
	}
	for _, c := range apperrCodes() {
		if !contract[c] {
			t.Errorf("apperr code %s missing from api/openapi.yaml ProblemCode", c)
		}
		delete(contract, c)
	}
	for c := range contract {
		t.Errorf("contract ProblemCode %s has no apperr.Code", c)
	}
}
