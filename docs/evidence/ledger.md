# Evidence Ledger

Every claim OpsGrid makes must link to proof: code, a test, a trace, a benchmark, a diagram or a document. A row is **done** only when its evidence link points at something that runs or can be inspected. Rows are filled gate by gate. `make proofs` regenerates [`proofs-report.md`](proofs-report.md) with pass/fail results and measured numbers.

Status legend: ⬜ not started · 🟨 in progress · ✅ proven

## Definition-of-Done matrix

| Property | Required proof | Gate | Status | Evidence |
|---|---|---|---|---|
| API contract conformance | Every API test response validated against `api/openapi.yaml`; generated code current | G0+ | 🟨 | [`apitest`](../../internal/api/apitest/contract.go) with [drift-rejection test](../../internal/api/apitest/contract_test.go); `make check-generated` |
| Tenant confidentiality | Automated cross-tenant access matrix passes for every tenant route | G1 | ⬜ | |
| Tenant relational integrity | Composite FK tests reject cross-org references | G1–G2 | ⬜ | |
| RLS fail-closed | Missing tenant context yields zero rows / rejected writes | G1 | 🟨 | Policy template proven on a probe table: [`TestRLSPolicyTemplateFailsClosed`](../../internal/platform/database/database_integration_test.go), [`TestTenantContextIsTransactionLocal`](../../internal/platform/database/database_integration_test.go). Real tenant tables arrive in G1. |
| Runtime role hardening | Schema test: runtime role is not owner and has no BYPASSRLS; every tenant table has RLS enabled + forced | G1 | 🟨 | Role attributes, no-DDL and timeouts proven: [`TestRuntimeRolesAreHardened`](../../internal/platform/database/database_integration_test.go). Per-table RLS schema test lands with the first tenant tables (G1). |
| Scheduling integrity | 100 concurrent conflicting assignments: 0 overlapping rows | G3 | ⬜ | |
| State-machine correctness | Exhaustive transition table + property tests | G2 | ⬜ | |
| Stale-write safety | Old versions rejected with 412; two-browser E2E | G4 | ⬜ | |
| API retry safety | 50 concurrent same-key requests produce 1 business operation | G4 | ⬜ | |
| Event durability | Committed event survives relay/worker outage | G6 | ⬜ | |
| Consumer replay safety | Duplicate stream delivery produces 1 business effect | G6 | ⬜ | |
| Worker recovery | Crash/restart recovers pending work (crash points A–D) | G6 | ⬜ | |
| Realtime convergence | Disconnected client reconnects to correct state | G5 | ⬜ | |
| Realtime backpressure | Slow consumer is disconnected at the bound; memory stays flat | G5 | ⬜ | |
| Authorization | Role × permission × object matrix tests | G1–G2 | ⬜ | |
| Auditability | Every command writes actor, entity, org, request and before/after | G2 | ⬜ | |
| Traceability | HTTP → DB → outbox → worker correlation shown in Tempo | G8 | ⬜ | |
| Reproducibility | Clean clone boots with one documented command | G0 | 🟨 | Verified locally from an empty state (`make reset && make dev`: healthy in 37 s, then the full suite green). The same command runs in CI's `e2e` job; first CI run on first push. Guide: [development.md](../development.md). |
| Migration safety | Build from empty + upgrade from the previous release snapshot; squawk clean | G0+ | 🟨 | Empty→head on every integration run; squawk clean; [migration file guards](../../db/db_test.go). Snapshot upgrade starts at the first release. |
| Security | Applicable ASVS 5.0.0 controls have implementation/test evidence | G9 | ⬜ | |
| Accessibility | WCAG 2.2 AA: automated axe + recorded manual keyboard/screen-reader pass | G2+ | 🟨 | Token contrast measured in both themes ([`tokens.test.ts`](../../apps/web/src/design/tokens.test.ts)); shell axe + focus tests ([`shell.test.tsx`](../../apps/web/src/app/shell.test.tsx)); E2E axe light/dark ([`smoke.spec.ts`](../../apps/web/e2e/smoke.spec.ts)). Manual pass pending. |
| Performance | Benchmark report with methodology, hardware and percentiles | G10 | ⬜ | |
| Operational readiness | Dashboards, alert rules and runbooks for each alert | G8 | ⬜ | |
| Supply chain | Signed images, SBOM, provenance attestation | G0+ | 🟨 | [`release.yml`](../../.github/workflows/release.yml): cosign keyless signing, SPDX SBOM attestation, SLSA provenance; SHA-pinned actions; pnpm 24 h release-age gate; OSV, govulncheck, Trivy, gitleaks in [`ci.yml`](../../.github/workflows/ci.yml). Runs on first push. |

## Gate log

| Gate | Closed | Tag | Notes |
|---|---|---|---|
| G0 Foundation | 2026-10-02 | `gate-0-foundation` | All exit criteria met locally: one-command boot, health endpoints, PostgreSQL/Redis reachable, migrations, telemetry reaching Tempo/Prometheus, spikes accepted (ADR-011, ADR-017), CI workflows written and validated (actionlint) with every CI command passing locally. Carried forward: the Keycloak login round trip (G1, needs the BFF) and the first GitHub Actions run (on first push). |
