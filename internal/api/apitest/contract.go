// Package apitest validates HTTP exchanges against the OpenAPI contract.
// Every API test routes through Conformance so the implementation cannot
// drift from api/openapi.yaml.
package apitest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	apispec "github.com/Tony5897/opsgrid/api"
)

var (
	loadOnce sync.Once
	router   routers.Router
	loadErr  error
)

func contractRouter() (routers.Router, error) {
	loadOnce.Do(func() {
		loader := openapi3.NewLoader()
		doc, err := loader.LoadFromData(apispec.OpenAPI)
		if err != nil {
			loadErr = err
			return
		}
		if err = doc.Validate(context.Background()); err != nil {
			loadErr = err
			return
		}
		router, loadErr = gorillamux.NewRouter(doc)
	})
	return router, loadErr
}

// Do serves req through h and fails the test if the request or the response
// does not conform to the contract. Undocumented routes fail too.
func Do(t testing.TB, h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rr, err := Check(h, req)
	if err != nil {
		t.Errorf("%v", err)
	}
	return rr
}

// Check serves req through h and returns an error describing any contract
// violation in the request or the response.
func Check(h http.Handler, req *http.Request) (*httptest.ResponseRecorder, error) {
	r, err := contractRouter()
	if err != nil {
		return nil, fmt.Errorf("load contract: %w", err)
	}
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	route, params, err := r.FindRoute(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s is not in the contract: %w", req.Method, req.URL.Path, err)
	}
	reqInput := &openapi3filter.RequestValidationInput{
		Request: req, PathParams: params, Route: route,
		Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
	}
	var errs []error
	if err := openapi3filter.ValidateRequest(context.Background(), reqInput); err != nil {
		errs = append(errs, fmt.Errorf("request violates contract: %w", err))
	}

	rr := httptest.NewRecorder()
	req.Body = io.NopCloser(bytes.NewReader(body))
	h.ServeHTTP(rr, req)

	respInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: reqInput,
		Status:                 rr.Code,
		Header:                 rr.Header(),
		Options:                &openapi3filter.Options{IncludeResponseStatus: true},
	}
	respInput.SetBodyBytes(rr.Body.Bytes())
	if err := openapi3filter.ValidateResponse(context.Background(), respInput); err != nil {
		errs = append(errs, fmt.Errorf("response %d violates contract: %w\nbody: %s", rr.Code, err, rr.Body.String()))
	}
	return rr, errors.Join(errs...)
}

// SchemaEnum returns the enum values of components.schemas[name].
func SchemaEnum(t testing.TB, name string) []string {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromData(apispec.OpenAPI)
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}
	ref, ok := doc.Components.Schemas[name]
	if !ok || ref.Value == nil {
		t.Fatalf("schema %s not in contract", name)
	}
	out := make([]string, 0, len(ref.Value.Enum))
	for _, v := range ref.Value.Enum {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("schema %s has a non-string enum value %v", name, v)
		}
		out = append(out, s)
	}
	return out
}
