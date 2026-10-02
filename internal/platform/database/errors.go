package database

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATE codes OpsGrid reacts to.
const (
	SQLStateUniqueViolation      = "23505"
	SQLStateForeignKeyViolation  = "23503"
	SQLStateCheckViolation       = "23514"
	SQLStateExclusionViolation   = "23P01"
	SQLStateNotNullViolation     = "23502"
	SQLStateSerializationFailure = "40001"
	SQLStateDeadlockDetected     = "40P01"
	SQLStateLockNotAvailable     = "55P03"
	SQLStateQueryCanceled        = "57014" // includes statement_timeout
	SQLStateInsufficientPriv     = "42501" // includes RLS WITH CHECK violations
)

// SQLState returns the PostgreSQL error code of err, or "".
func SQLState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// ConstraintName returns the violated constraint, or "".
func ConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// IsRetryable reports whether a transaction failed in a way that is safe to
// retry from the beginning.
func IsRetryable(err error) bool {
	switch SQLState(err) {
	case SQLStateSerializationFailure, SQLStateDeadlockDetected:
		return true
	}
	return false
}

// IsUniqueViolation reports a unique/PK violation, optionally on a specific
// constraint (empty matches any).
func IsUniqueViolation(err error, constraint string) bool {
	return is(err, SQLStateUniqueViolation, constraint)
}

// IsExclusionViolation reports an exclusion-constraint violation (for example
// overlapping technician assignments).
func IsExclusionViolation(err error, constraint string) bool {
	return is(err, SQLStateExclusionViolation, constraint)
}

// IsForeignKeyViolation reports an FK violation (including composite tenant
// FKs that reject cross-organization references).
func IsForeignKeyViolation(err error, constraint string) bool {
	return is(err, SQLStateForeignKeyViolation, constraint)
}

// IsNoRows reports pgx.ErrNoRows.
func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func is(err error, state, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != state {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}
