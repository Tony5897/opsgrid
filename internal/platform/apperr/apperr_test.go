package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
)

func TestCodeOf(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want apperr.Code
	}{
		{"typed", apperr.NotFound("work order"), apperr.CodeResourceNotFound},
		{"wrapped typed", fmt.Errorf("ctx: %w", apperr.Forbidden()), apperr.CodeForbidden},
		{"untyped is internal", errors.New(`pq: duplicate key value violates unique constraint "x"`), apperr.CodeInternal},
		{"nil-safe wrap", apperr.Wrap(errors.New("boom"), apperr.CodeDependencyUnavailable, "db down"), apperr.CodeDependencyUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := apperr.CodeOf(tc.err); got != tc.want {
				t.Fatalf("CodeOf() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestWithDetailsDoesNotMutateOriginal(t *testing.T) {
	t.Parallel()
	base := apperr.New(apperr.CodeScheduleConflict, "overlap")
	a := base.WithDetails("assignmentId", "a1")
	b := base.WithDetails("assignmentId", "b1")
	if base.Details != nil {
		t.Fatalf("base mutated: %v", base.Details)
	}
	if a.Details["assignmentId"] != "a1" || b.Details["assignmentId"] != "b1" {
		t.Fatalf("details crossed: a=%v b=%v", a.Details, b.Details)
	}
}

func TestUnwrapKeepsCause(t *testing.T) {
	t.Parallel()
	cause := errors.New("connection refused")
	err := apperr.Wrap(cause, apperr.CodeDependencyUnavailable, "database unavailable")
	if !errors.Is(err, cause) {
		t.Fatal("cause not reachable via errors.Is")
	}
}

func TestVersionConflictCarriesCurrentVersion(t *testing.T) {
	t.Parallel()
	err := apperr.VersionConflict(12)
	if err.Code != apperr.CodeVersionConflict || err.Details["currentVersion"] != int64(12) {
		t.Fatalf("unexpected: %+v", err)
	}
}
