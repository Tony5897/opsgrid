// Command genmatrix writes docs/security/permission-matrix.md and
// packages/api-client/permissions.json from the static authz role map.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Tony5897/opsgrid/internal/platform/authz"
)

func main() {
	root, err := repoRoot()
	if err != nil {
		fatal(err)
	}

	roles := []authz.Role{
		authz.RoleOwner,
		authz.RoleAdmin,
		authz.RoleDispatcher,
		authz.RoleTechnician,
		authz.RoleViewer,
	}
	byRole := map[string][]string{}
	for _, r := range roles {
		byRole[string(r)] = authz.PermissionStrings(r)
	}

	if err := writeJSON(filepath.Join(root, "packages", "api-client", "permissions.json"), byRole); err != nil {
		fatal(err)
	}
	if err := writeMarkdown(filepath.Join(root, "docs", "security", "permission-matrix.md"), roles, byRole); err != nil {
		fatal(err)
	}
}

func writeJSON(path string, byRole map[string][]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(byRole, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}

func writeMarkdown(path string, roles []authz.Role, byRole map[string][]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Permission matrix\n\n")
	b.WriteString("Generated from `internal/platform/authz`. Do not edit by hand — run `go generate ./internal/platform/authz/...`.\n\n")
	b.WriteString("Owner is allow-all (`*`): every catalog permission. `platform.operate` is owner-only (Platform Health / DLQ; IMPLEMENTATION_PLAN §4.9).\n\n")

	all := authz.AllPermissions()
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })

	b.WriteString("| Permission |")
	for _, r := range roles {
		fmt.Fprintf(&b, " %s |", r)
	}
	b.WriteString("\n|---|")
	for range roles {
		b.WriteString("---|")
	}
	b.WriteString("\n")

	for _, perm := range all {
		b.WriteString("| `")
		b.WriteString(string(perm))
		b.WriteString("` |")
		for _, r := range roles {
			ok := contains(byRole[string(r)], string(perm))
			cell := ""
			if ok {
				cell = "✓"
			}
			fmt.Fprintf(&b, " %s |", cell)
		}
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// go generate runs with cwd = package dir (internal/platform/authz).
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
		dir = parent
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "genmatrix: %v\n", err)
	os.Exit(1)
}
