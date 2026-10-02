package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
)

// WriteJSON writes v as a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// DecodeJSON strictly decodes a single JSON object from the request body
// into dst. It rejects unknown fields (no mass assignment), trailing data,
// non-JSON content types and bodies over the MaxBytes limit.
func DecodeJSON(r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mt, _, err := mime.ParseMediaType(ct)
		if err != nil || mt != "application/json" {
			return apperr.New(apperr.CodeUnsupportedMediaType, "Content-Type must be application/json.")
		}
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperr.Validation(apperr.FieldError{Field: "body", Message: "must contain a single JSON object"})
	}
	return nil
}

func decodeError(err error) error {
	var (
		syntaxErr   *json.SyntaxError
		typeErr     *json.UnmarshalTypeError
		maxBytesErr *http.MaxBytesError
	)
	switch {
	case errors.As(err, &maxBytesErr):
		return apperr.New(apperr.CodeRequestTooLarge, fmt.Sprintf("Request body exceeds %d bytes.", maxBytesErr.Limit))
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return apperr.Validation(apperr.FieldError{Field: "body", Message: "malformed JSON"})
	case errors.As(err, &typeErr):
		return apperr.Validation(apperr.FieldError{Field: typeErr.Field, Message: "has the wrong type"})
	case errors.Is(err, io.EOF):
		return apperr.Validation(apperr.FieldError{Field: "body", Message: "must not be empty"})
	default:
		// json: unknown field "x"
		return apperr.Validation(apperr.FieldError{Field: "body", Message: err.Error()})
	}
}
