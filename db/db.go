// Package db embeds the SQL migrations and bootstrap scripts so every binary
// and test harness applies exactly the reviewed files.
package db

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

//go:embed bootstrap/*.sql
var bootstrapFiles embed.FS

// Migrations returns the migrations directory as an fs.FS rooted at the
// directory itself (files at its top level).
func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err) // embedded path is static
	}
	return sub
}

// Bootstrap returns the contents of a bootstrap script (roles.sql, database.sql).
func Bootstrap(name string) (string, error) {
	b, err := bootstrapFiles.ReadFile(path.Join("bootstrap", name))
	return string(b), err
}

// LatestVersion returns the highest migration version embedded in the
// binary. Readiness checks compare it with the database to detect a process
// running against an un-migrated (or newer) schema.
func LatestVersion() (int64, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return 0, err
	}
	var latest int64
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(name, "_")
		if !ok {
			return 0, fmt.Errorf("migration %q lacks a version prefix", name)
		}
		v, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("migration %q: bad version: %w", name, err)
		}
		latest = max(latest, v)
	}
	return latest, nil
}
