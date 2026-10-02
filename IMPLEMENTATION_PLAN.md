# OpsGrid — Implementation Plan (build-ready)

## Context

The [architecture overview](docs/architecture/overview.md) describes **OpsGrid**. OpsGrid is a real-time, multi-tenant field-service platform (Go modular monolith + React/TS + PostgreSQL 18 + Redis 8 + Keycloak + OTel) built as a flagship portfolio project. The overview gives the *why* and the *what*: the architecture, the guarantees, the gates and the proof matrix. On its own it does not give the *how* at the level needed to start typing. It has no exact libraries, algorithms, schema/RLS mechanics, UX/design system, CI/supply-chain spec, or work packages with exit criteria. It also has a few technical gaps (listed in §1) that would cause bugs if built as written.

This plan turns the doc into an executable blueprint. Where the overview and this plan disagree, **this plan wins**, and each disagreement is recorded as an ADR. Nothing here weakens a guarantee from the source doc. Every change either makes a guarantee easier to enforce or adds one.

**Outcome:** a clean clone runs `make dev` and gets the full system. Each gate merges in a demonstrable state. The final README claim links every clause to code, a test, a trace, a benchmark or a doc.

---

## 0. First PR ("Gate 0 / PR-1: Blueprint")

Commit the planning artifacts before any code, so that the repo itself carries the reasoning:

- `IMPLEMENTATION_PLAN.md` (this plan) at the repo root
- `docs/architecture/overview.md`: the architecture overview (guarantees, threat/failure matrices, demo script) that this plan references as "the overview"
- `docs/adr/000-template.md` (MADR 4 format) and ADRs 001–021 (§1), each 1 page: Context / Decision / Consequences / Revisit-if
- `docs/evidence/ledger.md`: the doc's Definition-of-Done matrix with one empty "Evidence link" column per property. It is filled gate by gate.
- `docs/CONTRIBUTING.md`: engineering conventions (layering rules, "no handler logic", "every mutation = command spine", test commands, never weaken RLS/constraints to make a test pass), written as ordinary project conventions
- `README.md` stub using the narrative structure from the overview (no badge wall)
- `git init`, `main` protected, Conventional Commits, PR template with an **Evidence checklist** (tests added, ledger updated, ADR if decision, screenshots for UI)

---

## 1. Locked decisions (deltas and additions to the overview)

| # | Decision | Why / what it fixes |
|---|---|---|
| ADR-001…010 | As listed in the overview (modular monolith, Go, shared-schema MT, RLS, SQL-first, OIDC BFF, outbox, Redis Streams, WS event model, no K8s) | Unchanged |
| ADR-011 | **OpenAPI version chosen by a Gate 0 compatibility spike.** Try 3.2.1 → 3.1.x → 3.0.4 against the real generators (oapi-codegen strict-server, `@hey-api/openapi-ts`) and pick the highest that round-trips without hand patches | The doc flagged tool lag. This makes the call evidence-based and timeboxed (½ day) |
| ADR-012 | **Errors = RFC 9457 Problem Details** (`application/problem+json`) with extension members `code`, `requestId`, `traceId`, `details` | Standards-based. The doc's stable error codes are kept as `code` |
| ADR-013 | **Sessions stored in PostgreSQL**, not Redis (hashed session ID, encrypted refresh token) | The doc's failure matrix requires core sync operations to survive a Redis outage. Redis-backed sessions would log everyone out |
| ADR-014 | **Router = Go stdlib `net/http.ServeMux`** (method + wildcard patterns, `r.PathValue`). No chi | Fewer deps. The doc's `chi.URLParam` example becomes `r.PathValue("workOrderID")` |
| ADR-015 | **IDs = UUIDv7 via PostgreSQL 18 native `uuidv7()`**. Humans see per-org sequential numbers (`WO-1844`) from an `organization_counters` row updated in the same tx | Time-ordered, index-friendly PKs. Readable numbers without leaking global counts |
| ADR-016 | **Frontend stack** (§7.2): React 19 + React Compiler, Vite, TanStack Router/Query/Table/Virtual, Tailwind v4 + shadcn/ui (Radix), dnd-kit, Biome, Vitest browser mode, Storybook, MSW | Type-safe URL state, owned components, accessible DnD |
| ADR-017 | **Local S3-compatible server chosen by spike** (Garage or SeaweedFS; MinIO is avoided because of its 2025 distribution/maintenance changes). Requirements: presigned PUT/GET, CORS for browser PUT, Compose-friendly | Direct-upload flow must work locally exactly as on R2 |
| ADR-018 | **Logs: slog JSON to stdout is the source of truth.** It ships to Loki via the `otelslog` bridge if the OTel Go logs signal is stable by Gate 8; otherwise Grafana Alloy Docker discovery | Keeps the doc's caution but closes the "logs/Loki if desired" gap |
| ADR-019 | **CSRF: `http.CrossOriginProtection` (Go ≥1.25, Fetch-Metadata based) + `SameSite=Lax` `__Host-` cookie + required `X-OpsGrid-CSRF: 1` header on unsafe methods** | Defense-in-depth with no token plumbing |
| ADR-020 | **SPA served by the Go BFF** (`embed.FS`) in prod, so it is same-origin: no CORS and simple cookies. In dev, Vite proxies `/v1` and `/auth` to the API | Removes a whole class of cookie/CORS bugs |
| ADR-021 | **Event envelope follows CloudEvents 1.0 attribute names** (`id, source, type, subject, time, specversion, dataschema`) + `traceparent` extension | Standard shape, versionable, trace-linkable |

**Gaps in the overview fixed by this plan (implementers must follow these):**
1. **The RLS `current_setting` pitfall.** Once a tx-local setting has been used on a pooled connection, it reads back as `''`, not NULL, and `''::uuid` throws. All policies call `app.current_org_id()`, which returns `NULLIF(current_setting('app.organization_id', true), '')::uuid`, wrapped as `(SELECT app.current_org_id())` so it is evaluated once per statement (initplan).
2. **Duplicate scheduling state.** `work_orders.scheduled_start/end` duplicates `assignments`. It is replaced by `work_orders.requested_window tstzrange` (the customer's preferred window). Actual schedule = the active assignment. Invariant: **at most one active assignment per work order** (partial unique index).
3. **Retries on Redis Streams need a delay mechanism.** The doc says "backoff" without saying how. See §4.9 (ZSET retry scheduler).
4. **Realtime fan-out path was unspecified.** The relay publishes durable jobs to Streams *and* ephemeral notifications to Redis Pub/Sub. API nodes subscribe to Pub/Sub (§4.10).
5. **The idempotency "reserve" step** is implemented in the *same* transaction as the business write, so concurrent duplicates serialize on the unique index (§4.6).
6. **The outbox relay needs cross-tenant read without `BYPASSRLS`.** It gets a dedicated role with a role-targeted policy on `outbox_events` only (§5.1).
7. **Availability vs overlap.** The exclusion constraint guarantees *no overlap*. Working hours and time-off are v1 **soft warnings** (honest scope). PG18 `WITHOUT OVERLAPS` was considered and rejected because it cannot take the status predicate. This gets documented.

---

## 2. Toolchain & version policy

- **Pinned via `mise.toml`:** Go 1.27.x, Node 24 LTS, pnpm 12.x. Go dev tools are pinned with the **`tool` directive in `go.mod`** (`go tool sqlc`, `go tool oapi-codegen`, `go tool goose`, `go tool golangci-lint`), with no global installs.
- **Containers pinned by digest:** `postgres:18.6`, `redis:8.10`, `quay.io/keycloak/keycloak:26.7.4`, `otel/opentelemetry-collector-contrib`, `prom/prometheus`, `grafana/tempo`, `grafana/loki`, `grafana/grafana`, `axllent/mailpit`, `ghcr.io/shopify/toxiproxy`, plus the S3 server from ADR-017.
- **Renovate** keeps everything current: grouped weekly PRs, digest pinning, auto-merge only for patch-level dev deps with green CI.
- **Go libraries:** `jackc/pgx/v5` (+pgxpool), `sqlc`, `pressly/goose/v3` (SQL migrations, embedded), `redis/go-redis/v9`, `coder/websocket`, `coreos/go-oidc/v3` + `x/oauth2`, `oapi-codegen` (strict server, std-http), `aws-sdk-go-v2/s3` (presign; works with R2), `go.opentelemetry.io/otel` + `otelhttp` + `otelpgx` + `redisotel`, `johnfercher/maroto/v2` (PDF), `pgregory.net/rapid` (property tests), `testcontainers-go`, `stretchr/testify` (assert only), `go-faker/faker` (seeded).
- **Analysis:** golangci-lint v2 (`govet, staticcheck, errcheck, errorlint, gosec, revive, bodyclose, sqlclosecheck, noctx, contextcheck, exhaustive, depguard, gocritic`), `govulncheck`, **squawk** (Postgres migration safety linter), Redocly CLI (OpenAPI lint), **oasdiff** (breaking-change gate).

---

## 3. Repository blueprint (refines the overview's tree)

```text
opsgrid/
├── apps/web/                    # React SPA (pnpm workspace member)
│   └── src/{app,routes,features/*,components/ui,design,lib,realtime,test}
├── packages/api-client/         # GENERATED from api/openapi.yaml (hey-api: types, Zod schemas, TanStack Query options)
├── cmd/{api,worker,relay,seed}/ # thin mains: config → wire → run
├── internal/
│   ├── app/                     # composition root (wiring); only place that imports everything
│   ├── <domain>/                # tenancy, customers, sites, assets, workorders, scheduling,
│   │                            # notes, attachments, audit, notifications, reporting, realtime, idempotency
│   │   ├── domain.go            # types, state machine, invariants (pure; no IO)
│   │   ├── service.go           # application commands/queries (uses Repo interface + Tx)
│   │   ├── repo_pg.go           # sqlc-backed implementation
│   │   ├── http.go              # transport adapter (implements generated strict-server iface)
│   │   └── *_test.go
│   └── platform/{database,redisx,storage,telemetry,httpx,authn,authz,outbox,streams,config,clock}
├── db/{migrations,queries,seed}/ + sqlc.yaml
├── api/openapi.yaml             # source of truth (split files under api/paths, api/schemas, bundled in CI)
├── infra/{compose,keycloak/realm-opsgrid.json,otel,prometheus,grafana/dashboards,tempo,loki,toxiproxy}
├── test/{integration,e2e,load,failure,proofs}/
├── docs/{architecture,adr,security,runbooks,postmortems,benchmarks,evidence,plan}/
├── .github/{workflows,ISSUE_TEMPLATE,pull_request_template.md,CODEOWNERS}
├── compose.yaml  Dockerfile  Makefile  mise.toml  go.mod  pnpm-workspace.yaml  README.md
```

**Boundary rules, enforced by `depguard` in CI and not just by convention:**
- `internal/<domain>` must not import `net/http`, `platform/httpx`, or another domain's `repo_pg.go`. Cross-domain calls go through the other domain's exported service interface.
- `domain.go` files import only the stdlib plus `platform/clock`.
- Only `internal/app` and `cmd/*` may import everything.

---

## 4. Backend specification

### 4.1 Processes
`api` (HTTP + WS + serves SPA), `relay` (outbox → Streams/PubSub), `worker` (consumer groups + retry scheduler + cron-style jobs), `seed` (deterministic data). All four share config, telemetry and graceful-shutdown scaffolding. Migrations run as a one-shot `migrate` Compose service (goose binary, `opsgrid_migrator` creds) that the API `depends_on: service_completed_successfully`.

### 4.2 HTTP pipeline (outermost → innermost)
`recover → requestID (ULID, echo X-Request-Id) → otelhttp → slog access log → security headers (CSP, HSTS, XCTO, Referrer-Policy, Permissions-Policy, COOP) → body limit → CrossOriginProtection → session authn → org resolution (path {orgId} ↔ active membership) → rate limit (per user+org) → generated strict handler`. Problem+json mapping happens in one place: `httpx.WriteError(err)` maps typed domain errors to status and `code` and never leaks PG text.

### 4.3 Tenant transaction wrapper (`platform/database`)
```go
func (db *DB) InTenantTx(ctx context.Context, p authz.Principal, fn func(ctx context.Context, q *sqlc.Queries) error) error
// BEGIN (READ COMMITTED) → SELECT set_config('app.organization_id',$1,true), set_config('app.user_id',$2,true)
// → fn → COMMIT; retries ONLY on 40001/40P01 with jittered backoff (max 3); spans: db.tx
```
- No other code path may obtain a `pgx.Tx` for tenant tables. A `forbidigo` lint rule blocks `pool.Begin` outside `platform/database`.
- `InUserTx` is used for pre-tenant lookups (memberships, sessions) and sets only `app.user_id`.

### 4.4 Authorization (`platform/authz`)
- Roles → permissions are a **static Go map** (code-reviewed, testable). `go generate` emits `docs/security/permission-matrix.md` *and* `packages/api-client/permissions.json`, so the UI can hide what the server denies (UI hiding is never the enforcement).
- `Authorizer.Can(ctx, principal, perm, resource)`. `*_assigned` permissions resolve the relationship through a `ResourceLoader` (e.g. "is the technician on the active assignment?") inside the same tenant tx.
- The principal's membership is **re-read from the DB on every request** (an indexed PK lookup), so removed members lose access immediately (overview abuse case). WS connections re-check membership on a `membership.changed` event.
- Cross-tenant object access returns **404, not 403**, so existence doesn't leak.

### 4.5 The command spine (every mutation)
```text
handler → build Command (typed, no binding of arbitrary maps)
service.Execute(ctx, cmd):
  InTenantTx:
    1. idempotency.Claim(scope,key,fingerprint)   (§4.6) → replay? return stored response
    2. load aggregate FOR UPDATE (or version-predicated UPDATE)
    3. authz.Can(..., resource=aggregate)
    4. domain.Transition / validate invariants (pure)
    5. UPDATE ... WHERE version=$expected → 0 rows ⇒ ErrVersionConflict
    6. audit.Record(before,after redacted, request_id, actor, ua/ip metadata)
    7. outbox.Append(CloudEvent{..., traceparent})
    8. idempotency.Complete(status, body)
  COMMIT → return authoritative representation + ETag
```
The work-order state machine is a pure table in `workorders/domain.go`:

| From \ Cmd | Schedule | Unschedule | Dispatch | Start | Block | Resume | Complete | Cancel |
|---|---|---|---|---|---|---|---|---|
| draft | scheduled | – | – | – | – | – | – | cancelled |
| scheduled | scheduled (reassign) | draft | dispatched | – | – | – | – | cancelled |
| dispatched | dispatched (reassign) | draft | – | in_progress | – | – | – | cancelled |
| in_progress | – | – | – | – | blocked | – | completed | cancelled |
| blocked | – | – | – | – | – | in_progress | – | cancelled |
| completed / cancelled | terminal | | | | | | | |

The state machine is tested **exhaustively** (every from×cmd pair) and with **rapid** (random command sequences never reach an illegal state, terminal states stay terminal). `exhaustive` lint keeps the switch statements complete.

### 4.6 Idempotency (exact algorithm)
- Table PK `(organization_id, actor_user_id, operation, idempotency_key)`, columns `request_fingerprint bytea` (SHA-256 of method + route template + canonical JSON body), `state ('in_flight'|'completed')`, `response_status`, `response_body jsonb`, `expires_at` (24h).
- Step 1 runs **inside the business tx**: `INSERT ... ON CONFLICT DO NOTHING RETURNING`. A concurrent duplicate **blocks on the unique index** until the first tx commits or aborts. Postgres serializes them for free.
  - Row inserted → proceed. On rollback, the claim disappears with the tx, so a retry is safe.
  - Conflict → `SELECT` the row: fingerprint differs → `422 IDEMPOTENCY_CONFLICT`. Completed → replay the stored status/body with `Idempotent-Replayed: true`.
- `Idempotency-Key` is **required** on all POST commands (`400` if missing). The cleanup job deletes expired keys in batches.
- The client generates a key **per user intent** (when a form or drag starts) and reuses it across retries.

### 4.7 Optimistic concurrency
`ETag: "v{version}"` on GET. `If-Match` is **required** on PATCH and command endpoints (`428` if missing, `412 VERSION_CONFLICT` if stale). The 412 body includes the current representation so the UI can show a diff without another round trip.

### 4.8 Outbox relay
- `outbox_events(id uuidv7, organization_id, type, subject, aggregate_type, aggregate_id, payload jsonb, traceparent, created_at, published_at, attempts, last_error)`, plus a partial index `WHERE published_at IS NULL`.
- Loop: wake on `LISTEN outbox_new` (the tx does `pg_notify` on commit) **or** every 1s. Then `SELECT ... FOR UPDATE SKIP LOCKED LIMIT 100`, `XADD` to `stream:events` (MAXLEN ~ 1e6), `PUBLISH rt:org:{id}`, then `UPDATE published_at`. A crash between XADD and UPDATE causes a re-publish, which consumers dedupe (overview Case C).
- Housekeeping deletes published rows older than 7 days. Metrics: `outbox_unpublished`, `outbox_oldest_event_seconds`.

### 4.9 Workers (Redis Streams)
- Consumer group per handler (`cg:notifications`, `cg:exports`, …). `XREADGROUP BLOCK 5s COUNT 16`. A bounded goroutine pool per handler (semaphore).
- **Effect + dedupe in one PG tx:** `INSERT INTO processed_messages(consumer, message_id) ON CONFLICT DO NOTHING` → if it already exists, skip the effect. Otherwise do the effect in the same tx → commit → `XACK`. Non-DB side effects (email/storage) use deterministic keys (e.g. object key = `exports/{jobId}.csv`, email `Message-ID` = event id) so a repeat is harmless.
- **Retry:** on failure, `ZADD retry:{cg} score=now+backoff(attempt) member=envelope` then `XACK`. The scheduler goroutine moves due members back with `XADD` (Lua script for atomic pop+add). Backoff = `min(cap, base·2^n)` with full jitter. After N=8 attempts: `XADD stream:dlq` with error + attempts.
- **Stuck work:** `XAUTOCLAIM` messages idle for more than 60s (handles the case where a worker crashed before ACK).
- The DLQ is visible and re-drivable from the Platform Health UI (permission `platform.operate`, audited).
- Jobs: notification (Mailpit), CSV export, PDF service report (maroto), attachment finalize (HEAD object → size/type verify), overdue reminder (ticker plus leader election via `SET NX PX` lock), idempotency/outbox cleanup.

### 4.10 Realtime gateway (`coder/websocket`)
- `GET /v1/realtime`: session cookie authn, **Origin allowlist**, then upgrade. Client messages: `subscribe{topic}`, `unsubscribe`, `ping`. Server messages: `event{...}`, `subscribed`, `error`, `reconnect{reason}`.
- Topics: `org:{id}:operations`, `org:{id}:schedule`, `work-order:{id}`, `user:{id}:notifications`. Each subscribe runs `authz.Can`. Limits: 50 subs/conn, 5 conns/user, 32 KiB max message size.
- Per-connection **bounded send channel (256)**. If it is full, the connection is closed with code 4008 `slow_consumer`, the metric is incremented, and the client reconnects and refetches. Heartbeat: server ping every 20s, close after 2 missed pongs.
- Node fan-out: one `PSUBSCRIBE rt:org:*` per API node feeds a local hub. The hub routes by topic index. Events carry `{type, entityType, entityId, version, occurredAt, traceId}` and **never** the entity body.
- Presence: `SET presence:{org}:{user} EX 45`, refreshed every 20s, plus a per-WO "viewing" set with TTL. Ephemeral only.
- On shutdown the gateway sends `reconnect{reason:"server_shutdown"}`, then closes with 1012.
- Hub concurrency tests use **`testing/synctest`** (deterministic fake time for heartbeats and timeouts).

### 4.11 Identity & session (BFF)
- Keycloak realm `opsgrid` is imported from `infra/keycloak/realm-opsgrid.json` (demo users, confidential client `opsgrid-bff`, exact redirect URIs, PKCE S256 required, back-channel logout URL, brute-force detection on).
- Flow: `/auth/login` → state + nonce + PKCE verifier in a short-lived signed `__Host-oidc` cookie → Keycloak → `/auth/callback` validates state, nonce and ID token → upsert `users` by `(issuer, subject)` → create session → set the cookie `__Host-opsgrid_session` (Secure, HttpOnly, SameSite=Lax, Path=/).
- `sessions(id_hash sha256, user_id, created_at, last_seen_at, absolute_expires_at (12h), idle_expires_at (sliding 2h), refresh_token_enc (AES-256-GCM, key from env/KMS), ip, user_agent)`. The session ID rotates at login.
- `/auth/logout` deletes the session and does RP-initiated logout. `/auth/backchannel-logout` verifies the logout token and deletes the matching sessions.
- Follows RFC 9700 (OAuth 2.0 Security BCP): no implicit flow, no ROPC, exact redirects.
- Demo users: `dana@cascade` (dispatcher), `maria@cascade` and `james@cascade` (technicians), `owen@cascade` (owner), `nina@northstar` (dispatcher), and `alex` (**Cascade dispatcher + Northstar viewer**, to demo the org switcher). Stretch: passkeys via Keycloak WebAuthn passwordless for the owner account.

### 4.12 Attachments
`POST .../attachments/upload-intents` (authz `attachment.upload[_assigned]`, content-type allowlist, ≤25 MB) → row `state='pending'`, random key `org/{orgId}/wo/{woId}/{uuidv7}`, presigned PUT (5 min, content-length/type bound) → browser PUT → `POST .../attachments/{id}/complete` → worker verifies via HEAD → `state='available'` → realtime event. Downloads use presigned GET (60s) generated only after authz, and are never logged. Pending uploads older than 24h are swept.

### 4.13 Observability
- OTel SDK with resource attrs (`service.name`, `service.version`=git SHA, `deployment.environment`). Spans cover HTTP, pgx (otelpgx), Redis (redisotel), S3, and named domain spans (`WorkOrders.Complete`). The relay/worker **link** to the producer span via the stored `traceparent` (span links, not fake parentage).
- slog handler injects `trace_id`, `span_id`, `request_id`, `organization_id`, `actor_user_id`. A redaction `ReplaceAttr` drops keys matching `token|secret|password|cookie|authorization|signature|url_signed`.
- Metrics: the overview's full list plus `idempotency_replays_total`, `version_conflicts_total`, `schedule_conflicts_total`, `rls_context_missing_total`. Exemplars link metrics to traces.
- Dashboards are committed as JSON: System Overview, API (RED), PostgreSQL, Workers/Queues, Realtime, Tenancy & Security.
- Alerting rules (Prometheus) are committed for outbox age, DLQ>0, queue age, and 5xx rate. Each links to its runbook.

### 4.14 Config, health, shutdown
- Typed `config.Load()` from env, validated at boot, fail fast. Secrets only from env/files (`*_FILE` supported).
- Endpoints: `/livez` (process), `/readyz` (PG ping + migrations at the expected version; Redis is reported but **not** required for readiness), `/metrics` (internal port only).
- SIGTERM: readiness=false → `srv.Shutdown(30s)` → stop XREADGROUP → finish in-flight jobs (deadline) → WS reconnect broadcast → flush OTel → exit. `context` deadlines everywhere: HTTP 10s default, DB statement_timeout 5s for the app role, Redis 500ms.

---

## 5. Database specification

### 5.1 Roles & schemas
- `opsgrid_migrator` owns schema `app` and all tables.
- `opsgrid_app` (runtime): DML only, **no** ownership, **no** BYPASSRLS, `statement_timeout=5s`, `idle_in_transaction_session_timeout=10s`.
- `opsgrid_relay`: `SELECT/UPDATE` on `outbox_events` only, via a policy `TO opsgrid_relay USING (true)`.
- `opsgrid_worker`: same as app (sets tenant context per message from the envelope's org).
- `opsgrid_reporting_ro`: SELECT on reporting views with RLS.
- `REVOKE ALL ON SCHEMA public FROM PUBLIC`.
- `audit_events`: `REVOKE UPDATE, DELETE` from every runtime role, plus a trigger that raises on UPDATE/DELETE (belt and braces).

### 5.2 Conventions
`id uuid DEFAULT uuidv7()`. Every tenant table has `organization_id uuid NOT NULL`, `UNIQUE (organization_id, id)`, composite FKs to parents, `ENABLE` + `FORCE ROW LEVEL SECURITY`, and one policy generated from a template:
```sql
CREATE POLICY tenant_isolation ON app.<t> TO opsgrid_app, opsgrid_worker
  USING (organization_id = (SELECT app.current_org_id()))
  WITH CHECK (organization_id = (SELECT app.current_org_id()));
```
`timestamptz` everywhere. Status columns are `text` + `CHECK (status IN (...))` (cheaper to evolve than enums). `created_at/updated_at` (trigger), `version bigint NOT NULL DEFAULT 1` on mutable aggregates. A **schema test** (§8) asserts that every table with `organization_id` has RLS enabled+forced, a policy, and the composite unique key. This makes forgetting one a CI failure.

### 5.3 Migration sequence (goose, SQL, each reviewed + squawk-linted)
```text
00001 roles_schemas_extensions (btree_gist, pg_trgm, citext) + app.current_org_id()/current_user_id()
00002 identity: users(issuer,subject unique), sessions
00003 tenancy: organizations, locations, organization_memberships (policy: user_id = current_user_id() OR org = current_org_id()), organization_counters
00004 customers, customer_contacts
00005 sites (composite FK → customers), assets (composite FK → sites)
00006 technician_profiles (skills text[], home_location_id, color)
00007 work_orders (number per org UNIQUE(org, number), requested_window tstzrange, priority, status, version, search tsvector GENERATED)
00008 assignments + CHECK(ends_at > starts_at) + EXCLUDE overlap (overview) + partial UNIQUE(work_order_id) WHERE status IN active
00009 work_order_notes, attachments
00010 audit_events (append-only, monthly partitions via pg_partman-free simple scheme: declarative RANGE partitions created by worker job)
00011 outbox_events + NOTIFY trigger
00012 idempotency_keys, processed_messages
00013 report_jobs
00014 indexes justified by access patterns (each with a comment referencing the query name)
```
Expand-and-contract is mandatory for changes after Gate 4. CI applies migrations from empty **and** from the previous release's snapshot (§9).

### 5.4 Initial index plan (justify each in `docs/architecture/data-model.md`)
`work_orders (organization_id, status, priority, created_at DESC)`, `(organization_id, location_id, status)`, GIN on `search`, trigram GIN on `title`. `assignments` GiST (from the constraint) + `(organization_id, technician_user_id, starts_at)`. `audit_events (organization_id, entity_type, entity_id, created_at DESC)`. **Deliberately missing:** the index the Gate 10 intentional incident will "discover".

---

## 6. API contract

- `api/openapi.yaml` (split, bundled by Redocly) is the source of truth. CI regenerates the Go server interfaces + types and the TS client, then fails on `git diff`. `oasdiff breaking` against `main` fails on unflagged breaking changes.
- **Pagination:** keyset. `?limit≤100&cursor=` where the cursor is opaque base64url of `{sortKeyValues, id, sortSpec}` with an HMAC so tampering is detected. The response is `{items, nextCursor}`.
- **Filters/sort:** an explicit allowlist enum per collection. The `sort` param maps to prebuilt sqlc queries and is never interpolated.
- **Headers:** `Idempotency-Key`, `If-Match`/`ETag`, `X-Request-Id`, `Idempotent-Replayed`, `RateLimit` / `RateLimit-Policy` (IETF draft fields), `Retry-After` on 429/503.
- **Long-running:** report export returns `202` + `Location: /v1/organizations/{org}/report-jobs/{id}`, completion arrives over realtime, and the download is a presigned GET.
- Endpoint list as in the overview, plus: `GET /v1/me` (user + memberships + permissions per org), `POST .../work-orders/{id}/unschedule|block|resume`, `GET/POST .../work-orders/{id}/notes`, `POST .../attachments/{id}/complete`, `GET .../schedule?from&to&locationId` (lanes + busy intervals for conflict preview), `GET .../search?q=` (palette), `GET /v1/platform/health` (queues/outbox/realtime snapshot), `POST /v1/platform/dlq/{id}/redrive`.

---

## 7. Frontend, UX & design system

### 7.1 Users and what each needs
| Persona | Context | Primary jobs | UX implications |
|---|---|---|---|
| **Dispatcher** (Dana) | Desktop, often 2 monitors, all day | Triage the unassigned queue, assign without conflicts, react to changes | Dense, keyboard-first, live, zero-refresh, drag **and** keyboard assignment |
| **Technician** (Maria) | Phone, outdoors, gloves, spotty signal | See my day, start/complete, add notes/photos | Mobile-first "My Day", 48px targets, high contrast, offline-tolerant mutations |
| **Owner/Admin** (Owen) | Desktop, periodic | Team & permissions, reports, audit | Clarity, trust, exports |
| **Viewer** | Read-only | Status lookup | Same views with actions hidden |
| **Reviewer/Engineer** | Portfolio audience | See the machinery | Engineering Debug drawer, Platform Health |

### 7.2 Frontend architecture
- **React 19 + React Compiler** (no manual memo by default), **Vite** (current major), **TypeScript strict** (`noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`).
- **TanStack Router** (file-based, Zod-validated search params): filters, date, location and selected WO all live in the URL, so they are shareable, survive reload, and work with back/forward. Route loaders prefetch with `queryClient.ensureQueryData`.
- **TanStack Query v5:** the only home for server state. Query-key factory per feature (`workOrderKeys.detail(id)`). Generated `queryOptions` come from `packages/api-client`.
- **TanStack Table + TanStack Virtual** for lists. **React Hook Form + Zod** (generated Zod schemas reused for client validation). **dnd-kit** for the board. **cmdk** for the palette. **sonner** for toasts. **Recharts** for charts (themed via tokens). **Temporal API** (via `temporal-polyfill` where not native) for all timezone-correct scheduling math.
- **Org in the URL:** `/o/$orgSlug/...`. Two tabs can be in two orgs. Switching orgs clears org-scoped query caches.
- **Feature folders** (`features/dispatch`, `features/work-orders`, …) each own their routes' components, queries, mutations and tests. `components/ui` = shadcn primitives (owned code). `design/` = tokens.
- **API client wrapper:** `credentials:'include'`, auto `X-OpsGrid-CSRF`, problem+json → a typed `ApiError` with `code`. 401 → redirect to `/auth/login?returnTo=`.

### 7.3 Design system ("calm operations console")
- **Principles:** information first, color carries meaning (never decoration), every state is visible (loading, empty, error, stale, reconnecting, conflict), and live but never jumpy.
- **Tokens:** CSS variables in **OKLCH** in Tailwind v4 `@theme`, with semantic layers (`--surface-*`, `--text-*`, `--border-*`, `--status-{draft,scheduled,dispatched,in-progress,blocked,completed,cancelled}`, `--priority-{low,normal,high,urgent}`). **Light + dark** themes with matching contrast, both checked to WCAG 2.2 AA (4.5:1 text, 3:1 UI). Status is always **color + icon + label**.
- **Type:** Inter Variable (UI, `font-feature-settings: "tnum"` in tables/timelines) + JetBrains Mono (IDs, trace IDs). Modular scale 12/13/14/16/20/24/30.
- **Space & density:** 4px grid. **Comfortable/compact density toggle** (dispatchers want compact). Radius 6/8/12. Elevation via borders + subtle shadows, no heavy drop shadows.
- **Motion:** 120–200ms ease-out for state changes. Realtime-updated rows get a 1.2s background "pulse". All motion is gated by `prefers-reduced-motion`. Route transitions use the View Transitions API where supported.
- **Storybook** documents every primitive and composite in all states (light/dark × comfortable/compact), with the a11y addon on and MSW handlers for data-bound components.

### 7.4 App shell & IA
Left nav (collapsible, icon rail at <1280px): Overview · Dispatch · Schedule · Work Orders · Customers · Sites · Assets · Reports · Audit · Team · Settings · *Platform Health* (permission-gated). The top bar has: org switcher, location filter, ⌘K palette (navigate + search WOs/customers/assets + run actions like "Create work order"), **realtime status dot** (connected / reconnecting / offline, with a tooltip of the last event), presence avatars, user menu (theme, density, debug mode). Technicians land on **My Day** (bottom tab bar on mobile: My Day · Jobs · Notifications · Profile).

### 7.5 Flagship screen specs
1. **Live Dispatch Board** (`/o/$org/dispatch?date&location`)
   - Left column: Unassigned queue (sorted by priority then age, filter chips). Main area: technician **lanes** on a time axis (06:00–20:00 in *location* timezone, 15-min snap, now-line, virtualized rows).
   - **Drag** a WO card into a lane. While dragging, conflicting intervals (from `/schedule` busy intervals) shade red and the ghost shows the start/end time. On drop: optimistic placement (dashed outline "saving…") → server confirm → solid, or rollback + inline conflict toast with "Show conflicting job".
   - **Keyboard/single-pointer alternative** (WCAG 2.2 SC 2.5.7): select a card → `A` or "Assign…" opens a dialog with a technician combobox + time picker + live conflict preview. dnd-kit's keyboard sensor and screen-reader announcements are on too.
   - Remote changes animate in, an `aria-live="polite"` region announces throttled summaries ("WO-1844 assigned to Maria by Nina"), and avatars show who else is viewing.
2. **Work Order Detail**
   - Header: number, title, priority badge, status badge. **Only valid transitions render as buttons** (from `permissions.json` + the state machine table mirrored in TS and generated from Go to prevent drift).
   - Meta grid (customer · site · asset · assignee · window in site TZ). Tabs: Activity (merged timeline) · Notes · Attachments (drop zone + camera capture on mobile, upload progress, retry) · Asset History · Audit (permission-gated JSON diff viewer).
3. **Conflict resolution (412):** a modal shows a **three-column diff** (your change / current server / field) per changed field, with actions *Reapply my changes to latest* (new If-Match) · *Discard mine* · *Review*. This is the screen recording for the concurrency demo.
4. **Work Orders list:** TanStack Table with column visibility, saved views (URL presets), keyset pagination ("Load more" + virtual scroll), status/priority/tech/location/customer filters in the URL, and row live-pulse.
5. **My Day (technician, mobile):** chronological cards, a big Start/Complete button with confirm sheet, a quick note, and a photo button. **Offline-tolerant:** mutations persist through TanStack Query persisted paused mutations (IndexedDB) and replay on reconnect *with the same Idempotency-Key*, so a replay can't duplicate anything. An offline banner and a pending-sync badge per card show the state.
6. **Platform Health:** cards for Queue (pending, oldest age, retrying, DLQ), Outbox (unpublished, oldest age), and Realtime (connections, subs, reconnects/5m, slow-consumer drops), plus a DLQ table with inspect/redrive. It auto-refreshes every 5s and links to Grafana.
7. **Audit Log:** filter by actor/entity/action/date, request-ID search, before/after JSON diff, CSV export (async job).
8. **Team & Permissions:** member list, role select (audited), and a read-only **permission matrix** rendered from `permissions.json`.
9. **Engineering Debug drawer** (`Ctrl+Shift+D`, env-flagged): org, user, role, last request ID, entity + version, WS status, last event + measured latency (`occurredAt` → receipt), trace ID with an "Open in Grafana/Tempo" deep link.

### 7.6 Realtime client
One `RealtimeClient` (singleton, outside React) with exponential reconnect + jitter and `navigator.onLine` awareness. **On every (re)connect: invalidate all active org queries** (convergence guarantee). Topic subscriptions are driven by mounted routes (`useRealtimeTopic`). A declarative `eventType → queryKeys[]` invalidation table lives in `realtime/invalidation.ts`, and a version check skips invalidation if the cache already holds `version ≥ event.version`.

### 7.7 States, errors, feedback
Skeletons that match the final layout (no spinners for page loads). Route-level error boundaries with retry + request ID. Every problem `code` maps to human copy. A `RATE_LIMITED` toast shows a Retry-After countdown. Empty states teach the next action ("No unassigned work. Create a work order (C)").

### 7.8 Accessibility (target: WCAG 2.2 AA, verified)
Semantic landmarks, skip link, focus moves to the `h1` on route change, visible focus ring (2px, offset; never `outline:none`) meeting 2.4.11 Focus Not Obscured, target size ≥24px (2.5.8) and 48px on technician mobile, full keyboard operation including the board (2.5.7 alternative), `aria-live` for realtime, and forms with linked errors + `aria-invalid`. Checked with axe in Vitest/Storybook/Playwright, plus a manual VoiceOver pass per gate on the flagship flows.

### 7.9 Frontend performance & security budgets
- **Performance:** route-split bundles; initial JS ≤ 180 KB gzip; LCP ≤ 2.0s and INP ≤ 200ms on the local prod build; Lighthouse CI enforces the budgets. Fonts are self-hosted, preloaded and subset.
- **Security:** CSP `default-src 'self'; script-src 'self'; connect-src 'self' <s3-origin>; img-src 'self' data: blob: <s3-origin>; frame-ancestors 'none'`. No tokens in JS. No `dangerouslySetInnerHTML` (lint-banned). Notes render as plain text with autolinks.

---

## 8. Quality strategy (proof-driven)

| Layer | Tooling | Scope |
|---|---|---|
| Go unit/property | `go test`, rapid, fuzz (`FuzzTransition`, `FuzzCursorDecode`, `FuzzWSMessage`, `FuzzFingerprintCanonicalize`) | Domain, codecs |
| Go integration | testcontainers PG18 + Redis8. **Per-test DB cloned from a migrated template** (`CREATE DATABASE t_x TEMPLATE opsgrid_tpl`) for speed + isolation | Repos, RLS, constraints, idempotency, outbox |
| Schema invariants | `test/integration/schema_test.go` queries `pg_catalog` | Every tenant table: RLS forced + policy + composite key; runtime role not owner/BYPASSRLS |
| Tenant isolation matrix | Table-driven, generated from the OpenAPI route list, so **a new endpoint without a matrix entry fails CI** | 5 rows from the overview × every tenant route |
| Concurrency proofs | `test/proofs/`: 100-goroutine schedule race, 50× same-key POST, version race | Asserts the final DB invariant (0 overlaps, 1 WO, 1 audit) |
| Failure | `test/failure/` Go scenarios driving Docker + **Toxiproxy** (latency, reset, down) + process kill | The overview's 13-row failure matrix, one test each |
| Frontend unit/component | Vitest (browser mode) + Testing Library + MSW | Hooks, forms, invalidation table, conflict modal |
| Visual/a11y | Storybook + test-runner (axe) + Playwright screenshot diffs on key stories | Design system regressions |
| E2E | Playwright, **multi-context** (dispatcher + technician + 2nd dispatcher), against the full Compose stack | Assign-live, stale-write, offline-reconnect convergence, org switch isolation |
| Load | k6 (HTTP + `k6/websockets`), scenarios A–H from the overview, thresholds as budgets | Published in `docs/benchmarks/` with env + hardware |

`make proofs` runs every engineering proof and writes `docs/evidence/proofs-report.md` (timestamped, commit SHA, pass/fail + measured numbers). The ledger links to it.

---

## 9. CI/CD & supply chain (GitHub Actions)

- **PR workflow** (parallel jobs, `concurrency` cancel-in-progress, actions **pinned by SHA**, `permissions: read-all` by default):
  `go-lint` · `go-test` (unit + `-race`) · `go-fuzz-smoke` (30s per target) · `web-lint-typecheck` · `web-test` · `contract` (Redocly lint, codegen diff, oasdiff) · `migrations` (empty→head, prev-release-snapshot→head, squawk, schema invariants) · `integration` (services: postgres, redis) · `e2e` (Compose up, Playwright, upload traces/videos on failure) · `security` (govulncheck, osv-scanner on pnpm-lock, gitleaks, CodeQL go+ts) · `image` (build, Trivy scan, fail on HIGH/CRITICAL with fix available) · `lighthouse`.
- **Required checks** on `main`, CODEOWNERS, linear history, signed commits encouraged.
- **Main/release:** build multi-arch distroless `nonroot` images tagged by SHA → SBOM (syft) → **cosign keyless sign** → `actions/attest-build-provenance` (SLSA) → push GHCR. release-please produces the changelog + semver tags.
- **Deploy (demo):** manual `environment: demo` approval gate → migration job → deploy → smoke tests (`/readyz`, login round-trip, WS connect). Destructive migrations require a separate approval.
- OpenSSF Scorecard workflow, Dependabot security alerts, Renovate for updates.
- Dockerfile: multi-stage, `CGO_ENABLED=0`, `-trimpath -ldflags "-s -w -X version=$SHA"`, `USER nonroot`, read-only root FS in Compose, healthcheck.

---

## 10. Gate-by-gate execution plan

Each gate is a sequence of small PRs (≤400 lines changed where possible). A gate closes only when its **exit proofs** are green in CI and linked in `docs/evidence/ledger.md`. The **UX track runs in parallel from G0**, so the UI is never "added at the end". Sizes: S ≈ 1–3 days, M ≈ 1–2 weeks, L ≈ 2–4 weeks of focused work.

### G0 — Foundation (M)
1. PR-1 Blueprint (§0).
2. Toolchain: `mise.toml`, `go.mod` with tool directives, pnpm workspace, Biome, golangci-lint v2 config with depguard boundaries, `.editorconfig`.
3. Compose: `postgres`, `redis`, `keycloak` (realm import), `migrate`, `api`, `worker`, `relay`, `mailpit`, S3 server; profile `observability` (otel-collector, prometheus, tempo, loki, grafana); profile `chaos` (toxiproxy). Healthchecks + `depends_on` conditions.
4. Go skeleton: config, slog + OTel bootstrap, `httpx` pipeline, `/livez` `/readyz`, graceful shutdown, `platform/database` with `InTenantTx`, migration 00001.
5. **Spikes (timeboxed, each ends in an ADR):** OpenAPI version (ADR-011), S3 dev server (ADR-017).
6. Web skeleton: Vite + Router + Query + Tailwind v4 tokens + shadcn init + app shell (static) + Storybook + MSW + Vitest + Playwright smoke.
7. Makefile targets from the overview + `make proofs`, `make doctor` (checks tool versions, ports, Docker memory).
8. CI: PR workflow with lint/test/contract/migrations/image jobs.

**Exit:** the overview's foundation checklist ✓, plus `make doctor` passes on a clean machine, a trace is visible in Tempo, Storybook builds, and CI is green with required checks enabled.

### G1 — Identity & tenancy (L)
WP: migrations 00002–00003; OIDC BFF (§4.11); sessions; `/v1/me`; org resolution middleware; authz map + generator; RLS template + `app.current_org_id()`; schema-invariant test; seed with Cascade/Northstar/demo users; **UX:** login redirect, org switcher, permission-aware nav, user menu, theme/density toggles.
**Exit proofs:** the overview's 4 hostile statements as tests; removed member denied on the next request; RLS fail-closed when context is missing (`rls_context_missing` path returns zero rows/denies); runtime role is not owner/BYPASSRLS (schema test); session fixation (ID rotates on login); CSRF cross-site POST rejected.

### G2 — Core domain (L)
WP: migrations 00004–00007, 00009 (notes); customers/sites/assets CRUD; work orders with the state machine, audit, per-org numbering; keyset pagination + filters; search; **UX:** Work Orders list, Work Order Detail (header actions from the state machine), Customers/Sites/Assets pages, Create WO dialog (RHF+Zod, Idempotency-Key per intent), skeleton/empty/error states, ⌘K palette (navigate + search).
**Exit:** exhaustive + property state-machine tests; composite-FK cross-org insert rejected; every mutation writes audit (an integration test asserts audit rows per command); OpenAPI codegen diff clean; Storybook coverage for all new components.

### G3 — Scheduling (L) — *first systems showcase*
WP: migration 00006 + 00008; scheduling service (`Schedule/Unschedule/Reassign` within the WO state machine); `/schedule` endpoint with busy intervals; soft warnings (working hours, time-off); **UX:** Dispatch Board (lanes, unassigned queue, DnD + keyboard Assign dialog, conflict shading, optimistic placement + rollback), Schedule calendar (week view per tech).
**Exit:** `test/proofs/schedule_race`: 100 concurrent overlapping assignments → exactly the compatible winners commit and the `SELECT` overlap audit query returns 0. Board keyboard-only E2E passes. axe clean.

### G4 — Concurrency safety (M)
WP: ETag/If-Match on all mutable aggregates; 412 with the current representation; idempotency table + claim-in-tx (§4.6); cleanup job stub; **UX:** conflict-resolution diff modal; mutation retry policy (network errors retry with the same key; 4xx never).
**Exit:** 50× same-key concurrent POST → 1 WO, 1 audit, identical responses; different payload with the same key → 422; two-browser stale write → 412 + diff UI (Playwright); `docs/architecture/concurrency.md` written (isolation vs locks vs invariants vs versions vs idempotency).

### G5 — Realtime (L)
WP: gateway (§4.10) single node → Redis Pub/Sub multi-node (run 2 API replicas behind Caddy in Compose for the demo); presence; metrics; **UX:** RealtimeClient, invalidation table, status dot, live pulse, aria-live announcer, presence avatars, technician My Day live updates.
**Exit:** Playwright dispatcher→technician live assignment (<500ms p95 local); technician offline → server changes → reconnect → UI equals the DB (convergence); unauthorized topic subscribe → error + metric; slow-consumer test (synctest + integration) disconnects at the bound and memory stays flat; WS works across 2 API nodes.

### G6 — Async reliability (L)
WP: outbox (migration 00011 + LISTEN/NOTIFY), relay, Streams consumer groups, retry ZSET scheduler, XAUTOCLAIM, DLQ + redrive, processed_messages, notification worker (Mailpit), **UX:** Platform Health screen + DLQ table.
**Exit:** the overview's 7-item async checklist + outbox crash cases A–D as `test/failure/` scenarios (process kill at injected points via a `FAILPOINT` env hook compiled only under the `failpoints` build tag).

### G7 — Files & reports (M)
WP: storage interface + S3 impl; upload intents/complete/finalize worker; presigned downloads; report_jobs; CSV export + PDF service report; `report.ready` realtime; **UX:** attachments tab (drag-drop, mobile camera, progress, retry), Reports page (request → pending → ready toast → download), offline-persisted technician mutations (§7.5.5).
**Exit:** cross-tenant signed-download request → 404; size/type limits enforced by the presign policy; the full export flow E2E (HTTP→SQL→queue→worker→storage→realtime→audit); storage outage (Toxiproxy) → WO features unaffected and the attachment op fails clearly.

### G8 — Observability (M)
WP: full instrumentation (§4.13), span links across the outbox, dashboards + alert rules + runbooks for each alert, log pipeline per ADR-018, **UX:** Engineering Debug drawer with a Tempo deep link.
**Exit:** from a single request ID, find the trace HTTP→SQL→outbox→relay→worker in Grafana (recorded walkthrough in `docs/architecture/observability.md`); an injected error is diagnosable using only system output.

### G9 — Security (M) — *feature freeze*
WP: `docs/security/threat-model.md` (STRIDE per trust boundary + the overview's abuse cases), `asvs-verification.md` (ASVS 5.0.0 L2 applicable requirements → evidence link), header/CSP review, rate-limit tuning, redaction review, dependency + image scan triage, an adversarial test pass (IDOR fuzz across tenants using the generated route list, WS topic forgery, cursor tampering, idempotency key cross-user reuse, open-redirect on `returnTo`).
**Exit:** every abuse case has a linked passing test; ASVS sheet complete with N/A justifications; zero HIGH/CRITICAL unresolved findings.

### G10 — Performance & failure (M)
WP: perf seed profile (≈1M WOs, 5M audit rows) via `cmd/seed --profile=perf`; k6 scenarios A–H with thresholds; full failure matrix; race + fuzz long runs; **intentional incident** (the missing index from §5.4 and/or 5.9s storage latency via Toxiproxy) → diagnose with traces/EXPLAIN (ANALYZE, BUFFERS) → fix → re-benchmark → `docs/postmortems/` (labeled as a failure-injection exercise). Add a Redis cache **only** if a measurement justifies it (ADR).
**Exit:** `docs/benchmarks/results.md` with methodology, hardware, p50/p95/p99, error rate and resource graphs; all failure-matrix tests green.

### G11 — Portfolio release (M)
WP: polished demo seed (realistic names, history, photos), README in the overview's structure with every claim clause linked, architecture diagrams (C4 context/container as Mermaid + an exported SVG), a 5–8 min scripted demo video following the overview's 12 steps, an optional public demo (Neon/Upstash/R2 + a container host) with an honest "demo infra, no SLA" note, final ledger audit, v1.0.0 tag.
**Exit:** a fresh-machine clone → `make dev` → demo script runs end-to-end without manual fixes (verified on a second machine/VM). Every ledger row has evidence.

**Stretch (only after G11, per the overview):** passkeys, Kubernetes/Helm, IaC, tamper-evident audit hash chain, NATS/Kafka adapter behind `EventBus`, read replica, ABAC, an AI ops assistant over reporting views.

---

## 11. Risk register

| Risk | Likelihood / impact | Mitigation |
|---|---|---|
| Scope too large for the team's bandwidth | H/H | Gates are independently shippable. G0–G6 alone is already a top-tier portfolio. Cut stretch first, then G7 PDF, never proofs |
| OpenAPI 3.2 tool lag | M/M | ADR-011 spike in G0 |
| RLS performance pitfalls (per-row function calls) | M/M | `(SELECT fn())` initplan pattern, simple tenant-ID policies only, EXPLAIN checks in G10 |
| Keycloak config drift / complexity | M/M | Realm committed as JSON and imported at boot; no clicking in the admin UI |
| Redis Streams retry/claim subtleties | M/H | §4.9 explicit algorithm + failure tests with failpoints |
| Laptop resource pressure | M/M | Compose profiles; `make doctor` checks Docker memory; observability off by default |
| Accessible DnD complexity | M/M | Keyboard Assign dialog is the primary path built first; DnD is the enhancement |
| Version drift (Oct 2026 vs later) | H/L | Renovate + digest pins + `make doctor` |
| Free-tier changes | M/L | Local Compose is canonical; public demo is optional |

---

## 12. Verification

- **Plan-level:** after PR-1 the repo contains this plan, the ADRs, the evidence ledger and CONTRIBUTING.md. Every later PR updates the ledger or explains why not.
- **Per gate:** `make lint test test-race test-integration test-e2e` green locally and in CI. `make proofs` produces `docs/evidence/proofs-report.md` with the gate's proofs passing. A demo artifact (GIF/video/trace screenshot) goes in the PR.
- **System-level (G11):** on a clean VM, `git clone && make dev`, then run the overview's 12-step demo script by hand and via Playwright (`make demo-check`). Each step maps to an automated assertion: live update received, 409/412/422 behaviors, worker kill → backlog → recovery, trace found in Tempo, Northstar data disjoint.
- **Frontend quality:** Storybook a11y = 0 violations; Lighthouse CI budgets pass; a manual VoiceOver + keyboard-only run of Dispatch, WO Detail, My Day and the Conflict modal is recorded in `docs/evidence/a11y.md`.

---

## 13. Working agreements

- **Definition of Done (per PR):** typed contract updated (if API), tests at the right layer, no new lint suppressions without justification, ledger/ADR updated when relevant, UI PRs include Storybook stories + light/dark screenshots, migrations pass squawk + snapshot upgrade.
- **Never do this to get green:** disable RLS, give the runtime role ownership/BYPASSRLS, drop a constraint, skip `If-Match`/`Idempotency-Key`, or swallow errors. Each of these is a CI-detectable violation via the schema test, the contract, or lint.
- **Commit style:** Conventional Commits, scoped by module (`feat(scheduling): ...`). Squash-merge. A gate tag on close (`gate-3-scheduling`).
