package database_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Tony5897/opsgrid/internal/platform/database"
)

func pgErr(code, constraint string) error {
	return fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: code, ConstraintName: constraint})
}

func TestClassification(t *testing.T) {
	t.Parallel()
	excl := pgErr(database.SQLStateExclusionViolation, "assignments_no_technician_overlap")

	if !database.IsExclusionViolation(excl, "") || !database.IsExclusionViolation(excl, "assignments_no_technician_overlap") {
		t.Error("exclusion violation not detected")
	}
	if database.IsExclusionViolation(excl, "other") {
		t.Error("constraint filter ignored")
	}
	if database.IsUniqueViolation(excl, "") {
		t.Error("exclusion misclassified as unique")
	}
	if !database.IsForeignKeyViolation(pgErr(database.SQLStateForeignKeyViolation, "sites_customer_same_org_fk"), "sites_customer_same_org_fk") {
		t.Error("fk violation not detected")
	}
	if got := database.ConstraintName(excl); got != "assignments_no_technician_overlap" {
		t.Errorf("ConstraintName = %q", got)
	}
	if database.SQLState(errors.New("plain")) != "" {
		t.Error("non-pg error has sqlstate")
	}
	if !database.IsNoRows(fmt.Errorf("x: %w", pgx.ErrNoRows)) {
		t.Error("no rows not detected")
	}
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()
	for code, want := range map[string]bool{
		database.SQLStateSerializationFailure: true,
		database.SQLStateDeadlockDetected:     true,
		database.SQLStateUniqueViolation:      false,
		database.SQLStateQueryCanceled:        false,
	} {
		if got := database.IsRetryable(pgErr(code, "")); got != want {
			t.Errorf("IsRetryable(%s) = %v, want %v", code, got, want)
		}
	}
}
