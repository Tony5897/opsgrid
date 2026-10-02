# Developer guide

Everything needed to run, inspect, test and change OpsGrid locally. For *why* things are built this way, see the [architecture overview](architecture/overview.md), the [implementation plan](../IMPLEMENTATION_PLAN.md) and the [ADRs](adr/README.md).

---

## 1. Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Container runtime | Docker Engine 27+ with Compose v2 | OrbStack (recommended on macOS), Docker Desktop, or Colima. Give it ≥ 4 GB RAM (6 GB+ for the observability profile). |
| Go | 1.27.1 | Pinned in `go.mod`; with `GOTOOLCHAIN=auto` any Go ≥ 1.21 downloads it automatically. |
| Node.js | 24 LTS | |
| pnpm | 12.8.1 | |
| mise *(optional, recommended)* | latest | Installs exactly the Go/Node/pnpm versions in `mise.toml`: `mise install`. |

Check everything at once:

```bash
make doctor
```

It verifies the toolchain versions, that the Docker daemon is reachable, how much memory containers get, and that the ports OpsGrid uses are free.

## 2. First run

```bash
git clone https://github.com/Tony5897/opsgrid.git && cd opsgrid
make setup     # toolchain (mise), pnpm install, Playwright Chromium, Go modules
make dev       # builds the image and starts the core stack; waits until healthy
```

Then open **http://localhost:8080**.

`make dev` builds one container image containing all Go binaries plus the compiled web app, then starts the core services in dependency order. The order is: PostgreSQL → role bootstrap → migrations → API/worker/relay; Garage → bucket provisioning. It returns only when every health check passes.

## 3. What runs where

### Core stack (`make dev`)

| Service | URL / port | What it is | Credentials (local dev only) |
|---|---|---|---|
| **api** | http://localhost:8080 | Go API; serves the web app (same origin) | — |
| api (internal) | http://localhost:9090/livez, `/readyz` | Liveness/readiness (never public) | — |
| **worker** | internal :9090 | Background job consumers (handlers arrive in Gate 6) | — |
| **relay** | internal :9090 | Transactional-outbox publisher (Gate 6) | — |
| **migrate** | one-shot | Applies `db/migrations` as the schema owner, then exits | — |
| **postgres** | localhost:5432 | PostgreSQL 18.6, database `opsgrid` | runtime role `opsgrid_app` / `opsgrid-dev-app`; superuser `postgres` / `opsgrid-dev-superuser` |
| **redis** | localhost:6379 | Redis 8.10 (AOF persistence) | — |
| **keycloak** | http://localhost:8180 | OIDC identity provider; realm `opsgrid` | admin console: `admin` / `opsgrid-dev-admin` |
| **storage** | http://localhost:3900 | Garage, S3-compatible object storage; bucket `opsgrid` | see `.env.example` (`S3_*`) |
| **storage-init** | one-shot | Idempotently provisions Garage (layout, key, bucket, CORS) | — |
| **mailpit** | http://localhost:8025 | Catches all outgoing email | — |

Demo users exist in Keycloak's `opsgrid` realm (`owen`, `dana`, `maria`, `james`, `nina`, `alex`, all with password `opsgrid-dev-password`). Signing in to OpsGrid with them arrives in Gate 1. Today you can only sign in to Keycloak's account console at http://localhost:8180/realms/opsgrid/account.

### Observability (`make dev-obs`)

Starts everything above **plus** telemetry export:

| Service | URL | What to look at |
|---|---|---|
| **Grafana** | http://localhost:3000 | *Explore → Tempo* to search traces (service `opsgrid-api`); *Explore → Prometheus* for metrics, e.g. `opsgrid_http_server_request_duration_seconds_count`. No login needed locally. |
| Prometheus | http://localhost:9091 | Raw metrics and scrape targets |
| Tempo / Loki | (via Grafana) | Traces / logs storage |
| OTel Collector | localhost:4317 (gRPC), 4318 (HTTP) | Receives all telemetry |

Try it: `curl -s localhost:8080/v1/nope` returns a problem document with a `traceId`. Paste that ID into *Grafana → Explore → Tempo → TraceQL / Search* to see the request's trace.

### Failure injection (`docker compose --profile chaos up -d`)

Toxiproxy on http://localhost:8474 proxies PostgreSQL (15432), Redis (16379) and Keycloak (18080) so tests can inject latency and outages (Gate 10).

### Web development with hot reload

```bash
make dev        # backend services
make web-dev    # Vite on http://localhost:5173 (proxies /v1 and /auth to :8080)
```

Use **:5173** while editing UI code (instant hot reload) and **:8080** to see the production build exactly as it ships.

### Component workbench

```bash
make storybook  # http://localhost:6006
```

Every UI component in every state, with toolbar switches for **Theme** (light/dark/system) and **Density** (comfortable/compact), plus an **Accessibility** panel that runs axe live.

## 4. What you can see today (end of Gate 0)

- **http://localhost:8080:** pick a demo organization and you are in the operations console shell. Try:
  - the collapsible sidebar (state remembered per device)
  - the account menu (top right) → *Theme* and *Density*
  - narrowing the window below 768 px: navigation becomes a sheet
  - pressing <kbd>Tab</kbd> on page load: a *Skip to main content* link appears
  - the network indicator (turn Wi-Fi off to see *Offline*)

  Each area shows which gate delivers it.
- **http://localhost:8080/v1/meta:** build information from the contract-generated API.
- **http://localhost:8080/v1/anything-else:** an RFC 9457 problem document (`application/problem+json`) with `requestId` and `traceId`.
- **http://localhost:9090/readyz:** readiness of PostgreSQL, the schema version and Redis.
- **Keycloak** admin console → realm `opsgrid` → *Clients → opsgrid-bff*: PKCE S256 required, exact redirect URIs, back-channel logout.
- **Storybook:** the design system (color tokens, buttons, status badges, priorities, empty states, connection status).

## 5. Everyday commands

Run `make help` for the full list. The essentials:

| Command | What it does |
|---|---|
| `make dev` / `make dev-obs` | Start the stack (without / with observability) |
| `make logs` | Tail API, worker and relay logs (JSON, one line per event) |
| `make stop` | Stop containers, keep data |
| `make down` | Remove containers, keep volumes |
| `make reset` | **Delete all local data** (volumes) and containers |
| `make psql` | `psql` as `opsgrid_app`, the runtime role subject to Row-Level Security |
| `make migrate` | Re-run migrations |
| `make generate` | Regenerate Go server + TS types from `api/openapi.yaml` (and SQL later) |
| `make check` | Lint + race tests + integration tests + story tests (what CI runs before E2E) |
| `make test-e2e` | Playwright against the running stack |
| `make proofs` | Run the engineering proofs; writes `docs/evidence/proofs-report.md` |
| `make deps` | Report available dependency upgrades |
| `make secrets-scan` | gitleaks over the full git history |

## 6. Tests and what they prove

| Layer | Command | Where | Notes |
|---|---|---|---|
| Go unit (race detector) | `make test-race` | `**/*_test.go` | Includes the graceful-shutdown drain test and problem-mapping tests |
| Go integration | `make test-integration` | `*_integration_test.go` (`integration` build tag) | Starts PostgreSQL 18.6 via testcontainers. Proves RLS isolation, fail-closed tenant context, role hardening and retries |
| API contract | part of `make test-go` | `internal/api` | Every API response is validated against `api/openapi.yaml`; a test proves the harness rejects drift |
| Web unit/component | `make test-web` | `apps/web/src/**/*.test.ts(x)` | Runs in real Chromium. Includes **design-token contrast** measured from rendered pixels in both themes, and axe on the shell |
| Storybook | `make test-stories` | `*.stories.tsx` | Interaction tests + axe (violations fail) |
| E2E | `make test-e2e` | `apps/web/e2e` | Desktop + mobile; security headers, CSRF, SPA fallback, axe light/dark, keyboard |
| Spikes | `test/spikes/*/run.sh` | `test/spikes` | Reproduce ADR-011 (OpenAPI) and ADR-017 (object storage) evidence |

## 7. Repository map

```text
api/                    OpenAPI 3.1.1 contract (source of truth) + generator config
apps/web/               React app (routes, components, design tokens, tests, Storybook, E2E)
cmd/                    Process entrypoints: api, worker, relay, migrate, healthcheck
db/                     Migrations (goose), role/database bootstrap SQL, embedded at build
docs/                   Architecture, ADRs, evidence ledger, this guide
infra/                  Config for Compose services (Postgres init, Keycloak realm, Garage, OTel, Grafana…)
internal/api/           Generated server + contract-conformance test harness
internal/app/           Composition root (wires everything; the only package that imports all)
internal/platform/      Shared plumbing: config, database (tenant transactions), httpx, health, telemetry, redisx, webui
packages/api-client/    Typed fetch client + generated types/Zod schemas
scripts/                doctor, proofs, dependency report
test/spikes/            Reproducible technology evaluations
tools/                  Pinned Go tools (tools/go.mod) and the isolated OpenAPI TS generator
```

## 8. Making a change

1. Branch from `main`.
2. API change? Edit `api/openapi.yaml`, run `make generate`, then implement against the generated interface.
3. Schema change? Add `db/migrations/NNNNN_name.sql` (next number, no gaps). Start the Up section with `SET LOCAL lock_timeout` / `statement_timeout`. Run `make lint-sql test-integration`.
4. UI change? Add or extend a Storybook story for every state; check light and dark.
5. `make check` locally; open a PR and fill in the evidence checklist.

Conventions and the rules CI enforces are in [CONTRIBUTING](CONTRIBUTING.md).

## 9. Continuous integration

| Workflow | Trigger | What it does |
|---|---|---|
| `ci.yml` | PRs, `main` | Go lint/race/vuln · integration · web lint/typecheck/tests/stories/build budget · contract (Redocly, generated code, oasdiff) · squawk · gitleaks + OSV · image build + Trivy · full-stack E2E. Single required check: **CI passed** |
| `release.yml` | `main` | Multi-arch image to `ghcr.io/<owner>/opsgrid` (`sha-…`, `edge`), SPDX SBOM, cosign keyless signature, SLSA provenance |
| `codeql.yml` | PRs, `main`, weekly | Go + TypeScript security analysis |
| `scorecard.yml` | `main`, weekly | OpenSSF Scorecard |
| `deps-report.yml` | weekly | Updates the *Dependency report* issue (no automated commits, ADR-022) |

All actions are pinned to full commit SHAs, and every workflow defaults to read-only permissions.

**Verifying a released image.** `release.yml` signs every image (cosign, keyless/OIDC), attaches an SPDX SBOM attestation, and records SLSA build provenance. cosign's signature storage changed between v2 and v3 in a way that makes a v2 client unable to discover a v3-signed image's signature on GHCR (and vice versa) — use a **cosign v3+** client, matching what `sigstore/cosign-installer` installs in CI:

```bash
cosign verify ghcr.io/tony5897/opsgrid:edge \
  --certificate-identity "https://github.com/Tony5897/opsgrid/.github/workflows/release.yml@refs/heads/main" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

# or, via GitHub's own attestation store (no cosign needed):
gh attestation verify oci://ghcr.io/tony5897/opsgrid:edge -R Tony5897/opsgrid
```

**After pushing to GitHub for the first time**, in the repository settings:
- *Branches → Add rule for `main`*: require PRs, require status check **CI passed**, require linear history.
- *Security*: enable secret scanning, push protection and Dependabot **alerts** (alerts only, not version-update PRs).

## 10. Troubleshooting

| Symptom | Fix |
|---|---|
| `make doctor` reports Docker unreachable | Start OrbStack/Docker Desktop. On Apple Silicon, use the arm64 build of the runtime |
| A port is "in use" | Stop the other process, or `make down` if it's a previous OpsGrid run |
| `api` keeps restarting | `docker compose logs api`. Configuration errors are printed in full at startup |
| Keycloak slow to become healthy | First boot imports the realm (~30–60 s). It is capped at 768 MB RAM |
| Integration tests can't find Docker | `export DOCKER_HOST=$(docker context inspect --format '{{.Endpoints.docker.Host}}')` (the Makefile does this) |
| Need a clean slate | `make reset && make dev` |
| Browser tests fail to launch | `pnpm --filter @opsgrid/web exec playwright install chromium` |
