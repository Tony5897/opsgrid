# Evidence Ledger

Every claim OpsGrid makes must link to proof: code, a test, a trace, a benchmark, a diagram or a document. A row is **done** only when its evidence link points at something that runs or can be inspected. Rows are filled gate by gate. `make proofs` regenerates [`proofs-report.md`](proofs-report.md) with pass/fail results and measured numbers.

Status legend: ⬜ not started · 🟨 in progress · ✅ proven

## Definition-of-Done matrix

| Property | Required proof | Gate | Status | Evidence |
|---|---|---|---|---|
| Tenant confidentiality | Automated cross-tenant access matrix passes for every tenant route | G1 | ⬜ | |
| Tenant relational integrity | Composite FK tests reject cross-org references | G1–G2 | ⬜ | |
| RLS fail-closed | Missing tenant context yields zero rows / rejected writes | G1 | ⬜ | |
| Runtime role hardening | Schema test: runtime role is not owner and has no BYPASSRLS; every tenant table has RLS enabled + forced | G1 | ⬜ | |
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
| Reproducibility | Clean clone boots with one documented command | G0 | ⬜ | |
| Migration safety | Build from empty + upgrade from the previous release snapshot; squawk clean | G0+ | ⬜ | |
| Security | Applicable ASVS 5.0.0 controls have implementation/test evidence | G9 | ⬜ | |
| Accessibility | WCAG 2.2 AA: automated axe + recorded manual keyboard/screen-reader pass | G2+ | ⬜ | |
| Performance | Benchmark report with methodology, hardware and percentiles | G10 | ⬜ | |
| Operational readiness | Dashboards, alert rules and runbooks for each alert | G8 | ⬜ | |
| Supply chain | Signed images, SBOM, provenance attestation | G0+ | ⬜ | |

## Gate log

| Gate | Closed | Tag | Notes |
|---|---|---|---|
| G0 Foundation | | | |
