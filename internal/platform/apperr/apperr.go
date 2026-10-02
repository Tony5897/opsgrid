// Package apperr defines the typed errors that domain and application code
// return. It has no transport dependencies: platform/httpx maps a Code to an
// HTTP status and an RFC 9457 problem document (ADR-012).
package apperr

import (
	"errors"
	"fmt"
)

// Code is a stable, client-visible error identifier. Values are part of the
// public API contract and must never be renamed.
type Code string

const (
	CodeAuthenticationRequired Code = "AUTHENTICATION_REQUIRED"
	CodeForbidden              Code = "FORBIDDEN"
	CodeTenantNotFound         Code = "TENANT_NOT_FOUND"
	CodeResourceNotFound       Code = "RESOURCE_NOT_FOUND"
	CodeValidationFailed       Code = "VALIDATION_FAILED"
	CodePreconditionRequired   Code = "PRECONDITION_REQUIRED"
	CodeVersionConflict        Code = "VERSION_CONFLICT"
	CodeScheduleConflict       Code = "SCHEDULE_CONFLICT"
	CodeInvalidTransition      Code = "INVALID_TRANSITION"
	CodeIdempotencyConflict    Code = "IDEMPOTENCY_CONFLICT"
	CodeIdempotencyKeyRequired Code = "IDEMPOTENCY_KEY_REQUIRED"
	CodeRequestTooLarge        Code = "REQUEST_TOO_LARGE"
	CodeUnsupportedMediaType   Code = "UNSUPPORTED_MEDIA_TYPE"
	CodeRateLimited            Code = "RATE_LIMITED"
	CodeDependencyUnavailable  Code = "DEPENDENCY_UNAVAILABLE"
	CodeInternal               Code = "INTERNAL_ERROR"
)

// AllCodes lists every Code. Transport mappings are tested against it so a
// new code cannot ship without an HTTP status.
func AllCodes() []Code {
	return []Code{
		CodeAuthenticationRequired, CodeForbidden, CodeTenantNotFound, CodeResourceNotFound,
		CodeValidationFailed, CodePreconditionRequired, CodeVersionConflict, CodeScheduleConflict,
		CodeInvalidTransition, CodeIdempotencyConflict, CodeIdempotencyKeyRequired,
		CodeRequestTooLarge, CodeUnsupportedMediaType, CodeRateLimited,
		CodeDependencyUnavailable, CodeInternal,
	}
}

// FieldError describes one invalid input field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error is the typed error carried from domain code to the transport layer.
//
// Message is safe to show to clients. Err is the underlying cause and is
// logged but never serialized.
type Error struct {
	Code    Code
	Message string
	Details map[string]any
	Fields  []FieldError
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// New builds an Error with a client-safe message.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap attaches an internal cause to a client-safe error.
func Wrap(err error, code Code, message string) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

// WithDetails returns a copy of e with the given detail added.
func (e *Error) WithDetails(key string, value any) *Error {
	c := *e
	c.Details = make(map[string]any, len(e.Details)+1)
	for k, v := range e.Details {
		c.Details[k] = v
	}
	c.Details[key] = value
	return &c
}

// Convenience constructors for the most common cases.

func NotFound(resource string) *Error {
	return New(CodeResourceNotFound, resource+" not found")
}

func Forbidden() *Error {
	return New(CodeForbidden, "You do not have permission to perform this action.")
}

func Validation(fields ...FieldError) *Error {
	return &Error{Code: CodeValidationFailed, Message: "The request is invalid.", Fields: fields}
}

func VersionConflict(current int64) *Error {
	return New(CodeVersionConflict, "The resource changed since you loaded it.").
		WithDetails("currentVersion", current)
}

// CodeOf extracts the Code from err, returning CodeInternal for untyped
// errors. Untyped errors are always treated as internal so that their text
// never reaches a client.
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeInternal
}
