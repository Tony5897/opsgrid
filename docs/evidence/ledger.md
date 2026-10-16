# Evidence Ledger

Every claim OpsGrid makes must link to proof: code, a test, a trace, a benchmark, a diagram or a document. A row is **done** only when its evidence link points at something that runs or can be inspected. Rows are filled gate by gate. `make proofs` regenerates [`proofs-report.md`](proofs-report.md) with pass/fail results and measured numbers.

Status legend: ⬜ not started · 🟨 in progress · ✅ proven

## Definition-of-Done matrix

| Property | Required proof | Gate | Status | Evidence |
|---|---|---|---|---|
| API contract conformance | Every API test response validated against `api/openapi.yaml`; generated code current | G0+ | 🟨 | [`apitest`](../../internal/api/apitest/contract.go) with [drift-rejection test](../../internal/api/apitest/contract_test.go); `make check-generated` |
| Tenant confidentiality | Automated cross-tenant access matrix passes for every tenant route | G1 | 🟨 | Org probe: [`TestGetOrganizationMembershipGate`](../../internal/tenancy/tenancy_integration_test.go) (cross-tenant → TENANT_NOT_FOUND). `/v1/me` only own memberships: [`TestGetMeReturnsOnlyOwnMemberships`](../../internal/identity/identity_integration_test.go). Full per-route HTTP matrix grows with G2 domain routes. |
| Tenant relational integrity | Composite FK tests reject cross-org references | G1–G2 | ⬜ | Domain parent/child tables arrive in G2; locations already reject cross-org inserts under RLS in [`TestOrganizationsRLSFailsClosed`](../../internal/platform/database/tenancy_integration_test.go). |
| RLS fail-closed | Missing tenant context yields zero rows / rejected writes | G1 | ✅ | Probe table: [`TestRLSPolicyTemplateFailsClosed`](../../internal/platform/database/database_integration_test.go), [`TestTenantContextIsTransactionLocal`](../../internal/platform/database/database_integration_test.go). Real tables: [`TestOrganizationsRLSFailsClosed`](../../internal/platform/database/tenancy_integration_test.go), [`TestMembershipsVisibleInUserTx`](../../internal/platform/database/tenancy_integration_test.go). |
| Runtime role hardening | Schema test: runtime role is not owner and has no BYPASSRLS; every tenant table has RLS enabled + forced | G1 | ✅ | Role attributes: [`TestRuntimeRolesAreHardened`](../../internal/platform/database/database_integration_test.go). Per-table RLS + org-leading unique: [`TestSchemaTenantTablesHaveRLS`](../../internal/platform/database/tenancy_integration_test.go) (organizations, locations, memberships, counters). |
| Scheduling integrity | 100 concurrent conflicting assignments: 0 overlapping rows | G3 | ⬜ | |
| State-machine correctness | Exhaustive transition table + property tests | G2 | ⬜ | |
| Stale-write safety | Old versions rejected with 412; two-browser E2E | G4 | ⬜ | |
| API retry safety | 50 concurrent same-key requests produce 1 business operation | G4 | ⬜ | |
| Event durability | Committed event survives relay/worker outage | G6 | ⬜ | |
| Consumer replay safety | Duplicate stream delivery produces 1 business effect | G6 | ⬜ | |
| Worker recovery | Crash/restart recovers pending work (crash points A–D) | G6 | ⬜ | |
| Realtime convergence | Disconnected client reconnects to correct state | G5 | ⬜ | |
| Realtime backpressure | Slow consumer is disconnected at the bound; memory stays flat | G5 | ⬜ | |
| Authorization | Role × permission × object matrix tests | G1–G2 | 🟨 | Static map + `Can`: [`internal/platform/authz`](../../internal/platform/authz); matrix: [`docs/security/permission-matrix.md`](../security/permission-matrix.md); unit tests in `authz_test.go`. Object-level `*_assigned` loaders wait on G2. Removed-member re-read: [`TestGetOrganizationMembershipGate`](../../internal/tenancy/tenancy_integration_test.go). |
| BFF session authn | OIDC code+PKCE login; hashed session cookie; logout + back-channel; ID rotation | G1 | ✅ | Handlers + store: [`internal/platform/authn`](../../internal/platform/authn); RequireAuth: [`require.go`](../../internal/platform/authn/require.go) + [`require_test.go`](../../internal/platform/authn/require_test.go); lifecycle + ID rotation: [`store_integration_test.go`](../../internal/platform/authn/store_integration_test.go). Playwright Keycloak login green in CI E2E: [`smoke.spec.ts`](../../apps/web/e2e/smoke.spec.ts) ([81a74ee](https://github.com/Tony5897/opsgrid/actions/runs/38982631226)). |
| Auditability | Every command writes actor, entity, org, request and before/after | G2 | ⬜ | |
| Traceability | HTTP → DB → outbox → worker correlation shown in Tempo | G8 | ⬜ | |
| Reproducibility | Clean clone boots with one documented command | G0 | ✅ | Verified locally (`make reset && make dev`). Hosted CI `e2e` job boots the same Compose stack on `ubuntu-24.04` (green on `main`, e.g. [224f27c](https://github.com/Tony5897/opsgrid/actions/runs/38969773197)). Guide: [development.md](../development.md). |
| Migration safety | Build from empty + upgrade from the previous release snapshot; squawk clean | G0+ | 🟨 | Empty→head on every integration run; squawk clean through `00004`; [migration file guards](../../db/db_test.go). Snapshot upgrade starts at the first release. |
| Security | Applicable ASVS 5.0.0 controls have implementation/test evidence | G9 | ⬜ | |
| Accessibility | WCAG 2.2 AA: automated axe + recorded manual keyboard/screen-reader pass | G2+ | 🟨 | Token contrast measured in both themes ([`tokens.test.ts`](../../apps/web/src/design/tokens.test.ts)); shell axe + focus tests ([`shell.test.tsx`](../../apps/web/src/app/shell.test.tsx)); E2E axe light/dark ([`smoke.spec.ts`](../../apps/web/e2e/smoke.spec.ts)). Manual pass pending. |
| Performance | Benchmark report with methodology, hardware and percentiles | G10 | ⬜ | |
| Operational readiness | Dashboards, alert rules and runbooks for each alert | G8 | ⬜ | |
| Supply chain | Signed images, SBOM, provenance attestation | G0+ | 🟨 | [`release.yml`](../../.github/workflows/release.yml): cosign keyless signing, SPDX SBOM attestation, SLSA provenance; SHA-pinned actions; pnpm 24 h release-age gate; OSV, govulncheck, Trivy, gitleaks in [`ci.yml`](../../.github/workflows/ci.yml). Hosted runners restored 2026-10-16; required check **CI passed** on `main`. |

## Gate log

| Gate | Closed | Tag | Notes |
|---|---|---|---|
| G0 Foundation | 2026-10-02 | `gate-0-foundation` | All exit criteria met: one-command boot, health endpoints, PostgreSQL/Redis reachable, migrations, telemetry reaching Tempo/Prometheus, spikes accepted (ADR-011, ADR-017), CI green with required **CI passed**. Carried-forward Keycloak BFF + first hosted Actions run landed with G1 / 2026-10-16 reconnect. |
| G1 Identity & tenancy | 2026-10-16 | `gate-1-identity-tenancy` | Closed on [`81a74ee`](https://github.com/Tony5897/opsgrid/actions/runs/38982631226) (full CI incl. E2E + **CI passed**). Authz map, demo seed, `/v1/me`, org membership gate (cross-tenant → TENANT_NOT_FOUND), BFF sessions + Keycloak E2E, SPA shell. Object-level `*_assigned` and full per-route tenant matrix continue in G2. |
