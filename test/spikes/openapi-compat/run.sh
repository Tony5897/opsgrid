#!/usr/bin/env bash
# Re-runs the ADR-011 compatibility spike: generates Go (oapi-codegen, pinned
# in tools/go.mod) and TypeScript (hey-api, pinned in packages/api-client)
# from the sample contract at several OpenAPI versions, with and without
# 3.1-style nullable types, and compiles the Go output in a scratch module.
set -euo pipefail
root=$(cd "$(dirname "$0")/../../.." && pwd)
here="$root/test/spikes/openapi-compat"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export GOTOOLCHAIN=go1.27.1 GOFLAGS=-mod=mod

printf 'module spike\n\ngo 1.27.1\n' >"$work/go.mod"
printf '| Variant | oapi-codegen | hey-api |\n|---|---|---|\n'
for v in 3.0.4 3.1.1 3.2.1; do
  for nullable in no yes; do
    name="v${v//./_}_$nullable"
    spec="$work/$name.yaml"
    sed "s/__VERSION__/$v/" "$here/spec.tmpl.yaml" >"$spec"
    if [[ $nullable == yes ]]; then
      [[ $v == 3.0.4 ]] && continue # type arrays are not valid 3.0
      python3 - "$spec" <<'PY'
import sys; p=sys.argv[1]; t=open(p).read()
t=t.replace('        description: { type: string }\n','        description: { type: [string, "null"] }\n        assignedTo: { type: [string, "null"], format: uuid }\n',1)
open(p,'w').write(t)
PY
    fi
    mkdir -p "$work/$name"
    printf 'package: %s\ngenerate: { std-http-server: true, strict-server: true, models: true }\noutput: %s\n' \
      "$name" "$work/$name/gen.go" >"$work/$name.cfg.yaml"
    go_result="❌"
    if (cd "$root" && go tool -modfile=tools/go.mod oapi-codegen -config "$work/$name.cfg.yaml" "$spec") >/dev/null 2>&1 \
      && (cd "$work" && go mod tidy >/dev/null 2>&1 && go vet "./$name/" >/dev/null 2>&1); then go_result="✅"; fi
    ts_result="❌"
    if (cd "$root" && pnpm --filter @opsgrid/openapi-gen exec openapi-ts -i "$spec" -o "$work/ts-$name" -p @hey-api/typescript) >/dev/null 2>&1 \
      && [[ -f "$work/ts-$name/types.gen.ts" ]]; then ts_result="✅"; fi
    printf '| %s%s | %s | %s |\n' "$v" "$([[ $nullable == yes ]] && echo ' + type: [T, null]')" "$go_result" "$ts_result"
  done
done
