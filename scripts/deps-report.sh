#!/usr/bin/env bash
# Prints a Markdown report of available dependency upgrades. Read-only:
# it never modifies files. Used by `make deps` and the weekly workflow.
set -uo pipefail
cd "$(dirname "$0")/.."
export GOTOOLCHAIN=${GOTOOLCHAIN:-go1.27.1}

echo "# Dependency report"
echo
echo "Generated $(date -u +%Y-%m-%dT%H:%M:%SZ). Upgrades are applied and committed by the maintainer (ADR-022); pnpm refuses releases younger than 24 hours."
echo
echo "## Go modules (direct)"
echo
echo '```'
go list -u -m -f '{{if and (not .Indirect) .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all 2>/dev/null | sed '/^$/d' || true
echo '```'
echo
echo "## Go tools (tools/go.mod)"
echo
echo '```'
go list -modfile=tools/go.mod -u -m -f '{{if and (not .Indirect) .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all 2>/dev/null | sed '/^$/d' || true
echo '```'
echo
echo "## JavaScript (pnpm)"
echo
echo '```'
pnpm -r outdated 2>/dev/null || true
echo '```'
echo
echo "## Container images"
echo
echo "Pinned in compose.yaml and Dockerfile:"
echo
echo '```'
grep -hoE 'image: [^ ]+' compose.yaml | sort -u
grep -hE '^ARG (GO|NODE)_VERSION|^FROM' Dockerfile
echo '```'
