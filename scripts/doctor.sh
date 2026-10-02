#!/usr/bin/env bash
# Verifies the local environment can build and run OpsGrid.
set -uo pipefail

ok=0; warn=0; fail=0
pass()  { printf '  \033[32m✓\033[0m %s\n' "$1"; ok=$((ok+1)); }
note()  { printf '  \033[33m!\033[0m %s\n' "$1"; warn=$((warn+1)); }
bad()   { printf '  \033[31m✗\033[0m %s\n' "$1"; fail=$((fail+1)); }

ver_ge() { [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -1)" = "$2" ]; }

echo "Toolchain"
if command -v go >/dev/null; then
  gv=$(GOTOOLCHAIN=auto go env GOVERSION 2>/dev/null | sed 's/^go//')
  if ver_ge "$gv" "1.27.1"; then pass "go $gv"; else note "go $gv (go.mod requests 1.27.1; GOTOOLCHAIN=auto will download it)"; fi
else bad "go not found (https://go.dev/dl)"; fi

if command -v node >/dev/null; then
  nv=$(node -v | sed 's/^v//')
  if ver_ge "$nv" "24.0.0"; then pass "node $nv"; else bad "node $nv (need >= 24 LTS)"; fi
else bad "node not found"; fi

if command -v pnpm >/dev/null; then pass "pnpm $(pnpm -v)"; else bad "pnpm not found (npm i -g pnpm)"; fi

echo "Containers"
if command -v docker >/dev/null && docker info >/dev/null 2>&1; then
  pass "docker $(docker version --format '{{.Server.Version}}' 2>/dev/null)"
  if docker compose version >/dev/null 2>&1; then pass "$(docker compose version --short | sed 's/^/compose /')"; else bad "docker compose plugin missing"; fi
  mem=$(docker info --format '{{.MemTotal}}' 2>/dev/null || echo 0)
  gib=$((mem / 1024 / 1024 / 1024))
  if [ "$gib" -ge 6 ]; then pass "container memory ${gib} GiB"
  elif [ "$gib" -ge 4 ]; then note "container memory ${gib} GiB (core stack OK; observability profile may be tight)"
  else bad "container memory ${gib} GiB (need >= 4)"; fi
else bad "docker daemon not reachable (install/start OrbStack, Colima or Docker Desktop)"; fi

echo "Ports"
for p in 5432 6379 8080 8180 9090 3900 5173 8025; do
  if lsof -nP -iTCP:"$p" -sTCP:LISTEN >/dev/null 2>&1; then
    owner=$(lsof -nP -iTCP:"$p" -sTCP:LISTEN | awk 'NR==2{print $1}')
    case "$owner" in
      com.docke*|OrbStack*|docker*|vpnkit*|limactl*|node*) pass "port $p in use by $owner (likely OpsGrid)";;
      *) note "port $p in use by $owner";;
    esac
  else pass "port $p free"; fi
done

echo
printf 'doctor: %d ok, %d warnings, %d failures\n' "$ok" "$warn" "$fail"
[ "$fail" -eq 0 ]
