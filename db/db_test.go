package db_test

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/Tony5897/opsgrid/db"
)

var migrationName = regexp.MustCompile(`^\d{5}_[a-z0-9_]+\.sql$`)

func TestMigrationFilesAreWellFormed(t *testing.T) {
	t.Parallel()
	entries, err := fs.ReadDir(db.Migrations(), ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no migrations embedded")
	}
	seen := map[string]bool{}
	for i, e := range entries {
		name := e.Name()
		if !migrationName.MatchString(name) {
			t.Errorf("%s: must match NNNNN_snake_case.sql", name)
		}
		version := name[:5]
		if seen[version] {
			t.Errorf("duplicate version %s", version)
		}
		seen[version] = true
		// Versions are contiguous: 00001, 00002, ...
		if want := i + 1; version != padded(want) {
			t.Errorf("%s: expected version %s (no gaps)", name, padded(want))
		}
		body, err := fs.ReadFile(db.Migrations(), name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "-- +goose Up") {
			t.Errorf("%s: missing '-- +goose Up'", name)
		}
		// goose splits on semicolons; dollar-quoted bodies (functions, DO
		// blocks) must be wrapped so they are sent as one statement.
		if n := strings.Count(string(body), "$$") / 2; n > 0 {
			begins := strings.Count(string(body), "-- +goose StatementBegin")
			ends := strings.Count(string(body), "-- +goose StatementEnd")
			multiline := strings.Count(string(body), "$$\n") // single-line SQL functions are safe
			if begins != ends || begins < multiline/2 {
				t.Errorf("%s: dollar-quoted blocks need -- +goose StatementBegin/End (begins=%d ends=%d)", name, begins, ends)
			}
		}
	}
}

func padded(n int) string { return fmt.Sprintf("%05d", n) }

func TestLatestVersion(t *testing.T) {
	t.Parallel()
	v, err := db.LatestVersion()
	if err != nil {
		t.Fatal(err)
	}
	entries, _ := fs.ReadDir(db.Migrations(), ".")
	if v != int64(len(entries)) {
		t.Fatalf("LatestVersion = %d, want %d", v, len(entries))
	}
}

func TestBootstrapScriptsEmbedded(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"roles.sql", "database.sql"} {
		s, err := db.Bootstrap(name)
		if err != nil || s == "" {
			t.Fatalf("%s: %v", name, err)
		}
	}
	roles, _ := db.Bootstrap("roles.sql")
	if strings.Contains(strings.ToUpper(roles), "PASSWORD '") {
		t.Fatal("roles.sql must not contain passwords")
	}
}
