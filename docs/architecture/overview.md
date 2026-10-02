# OpsGrid — Architecture Overview

OpsGrid is a real-time, multi-tenant field-service and asset-operations platform. Organizations use it to manage locations, customers, sites, equipment, technicians, work orders, schedules, files and operational events.

The domain is deliberate. Field operations force the problems that serious business software has: tenant isolation, nuanced authorization, relational integrity, concurrent editing, scheduling conflicts, real-time propagation, durable asynchronous work, duplicate delivery, file storage, auditability, migrations, standards-based identity, distributed tracing, failure recovery and measured performance.

> **Enterprise-grade does not mean "use the most infrastructure." It means the system has explicit, demonstrable guarantees.**

The build plan is in [`IMPLEMENTATION_PLAN.md`](../../IMPLEMENTATION_PLAN.md). Decisions are recorded in [`docs/adr/`](../adr/).

---

## 1. Questions the system must answer with evidence

1. Can Tenant A ever retrieve Tenant B's work orders?
2. What happens if two dispatchers edit the same job simultaneously?
3. Can one technician be scheduled in two places at once?
4. What happens if a worker crashes halfway through a job?
5. What happens when the same API command arrives twice?
6. What happens when a WebSocket disconnects during an update?
7. Can we identify which user changed an asset three weeks ago?
8. Can a failed request be traced from the browser through SQL and through an asynchronous worker?
9. Can a developer clone the repository and reproduce the system?

Each answer is backed by an automated test, a database invariant, a trace or a document. The links are tracked in [`docs/evidence/ledger.md`](../evidence/ledger.md).

## 2. System shape

A **modular monolith** with separately deployable API, relay and worker processes. It is deliberately not microservices: cross-domain invariants stay inside ordinary PostgreSQL transactions, and the module boundaries keep later extraction credible.

```text
                              ┌──────────────────────┐
                              │   Keycloak (OIDC)    │
                              └──────────┬───────────┘
                                         │ Authorization Code + PKCE
┌──────────────────┐  HTTPS/REST  ┌──────▼───────────────┐
│  React / TS SPA  │─────────────▶│   Go API / BFF       │
│                  │  WebSocket   │   (serves the SPA)   │
└──────────────────┘─────────────▶└──┬───────┬───────┬───┘
                                     │       │       │
                     ┌───────────────┘       │       └──────────────┐
                     ▼                       ▼                      ▼
             ┌──────────────┐        ┌──────────────┐       ┌────────────────┐
             │ PostgreSQL   │        │ Redis        │       │ Object storage │
             │ domain state │        │ Streams      │       │ (S3 API / R2)  │
             │ RLS · audit  │        │ Pub/Sub      │       └────────────────┘
             │ outbox       │        │ presence     │
             └──────┬───────┘        │ rate limits  │
                    │ outbox         └──────┬───────┘
                    ▼                       │ jobs
             ┌──────────────┐               ▼
             │ Outbox relay │──────▶ ┌──────────────┐
             └──────────────┘        │  Go workers  │
                                     └──────────────┘
          all processes ──▶ OpenTelemetry Collector ──▶ Prometheus · Tempo · Loki ──▶ Grafana
```

**Principle:** HTTP plus the transactional state in PostgreSQL is authoritative. WebSocket events only tell clients that authoritative state changed. Clients then refetch.

## 3. Domain modules

| Module | Responsibility |
|---|---|
| Identity | External identity linkage, user profile, sessions |
| Tenancy | Organizations, locations, memberships, permissions |
| Customers | Customer companies and contacts |
| Sites | Physical customer locations |
| Assets | Equipment at customer sites |
| Work orders | Operational work lifecycle (state machine) |
| Scheduling | Technician assignments and time windows |
| Notes | Work-order discussion |
| Attachments | Photos and documents via signed direct upload |
| Audit | Append-only record of sensitive changes |
| Notifications | Asynchronous operational messages |
| Realtime | Authenticated live event subscriptions |
| Reporting | Dashboards and exports |
| Platform | Database, Redis, storage, telemetry, HTTP plumbing |

## 4. Demo tenants

```text
Cascade Facilities            Northstar Mechanical
├── Portland                  ├── Seattle
├── Salem                     └── Tacoma
└── Eugene
```

The two tenants are fully independent. One demo user (`alex`) belongs to both, as a Cascade dispatcher and a Northstar viewer. Same identity, different authority.

## 5. Tenant protection: three layers, no single point of trust

| Layer | Question it answers |
|---|---|
| **A. Application authorization** | Is the caller a member of this org, do they hold the permission, and may they act on this specific object? |
| **B. Tenant-aware relational constraints** | Can this row even *reference* another organization's row? (Composite `(organization_id, id)` foreign keys.) |
| **C. PostgreSQL Row-Level Security** | Even if a query forgets `WHERE organization_id = …`, may this session see the row? (`FORCE ROW LEVEL SECURITY`, runtime role is not the owner and has no `BYPASSRLS`, tenant context is transaction-local.) |

## 6. Roles and permissions (initial)

| Role | Permissions |
|---|---|
| owner | `*` |
| admin | organization.read, member.read, member.manage, customer.\*, asset.\*, work_order.\*, schedule.\*, audit.read, report.read |
| dispatcher | customer.read, asset.read, work_order.read/create/update/assign, schedule.read/manage, report.read |
| technician | customer.read_assigned, asset.read_assigned, work_order.read_assigned/start_assigned/complete_assigned/note_assigned, attachment.upload_assigned |
| viewer | customer.read, asset.read, work_order.read, schedule.read |

Authorization goes through one `Authorizer.Can(ctx, principal, permission, resource)` interface. Role names are never compared in business code.

## 7. Abuse cases (each must have a passing test)

1. The attacker changes the organization UUID in the URL.
2. The attacker guesses another tenant's work-order UUID.
3. A technician subscribes to an admin-only WebSocket topic.
4. A user removed from an organization keeps an existing browser tab open.
5. A duplicate command is delivered during a client retry.
6. A duplicate queue message arrives after a worker restart.
7. A user requests a signed download for another tenant's attachment.
8. A slow WebSocket client causes server buffer growth.
9. A compromised worker attempts a tenant-crossing query.
10. The application starts without tenant context.
11. A support/admin user accesses a tenant record.

## 8. Failure matrix

| Failure | Expected behavior |
|---|---|
| PostgreSQL unavailable | Mutations fail quickly and clearly; no fake success |
| Redis unavailable | Synchronous DB operations persist correctly; async/realtime degradation is visible |
| Worker killed | Pending jobs remain recoverable |
| Outbox relay killed | Unpublished events accumulate; lag is visible and alerted |
| Keycloak unavailable | New logins fail; existing sessions follow the documented expiry design |
| Object storage unavailable | Work orders keep working; attachment operations fail separately |
| WebSocket dropped | UI shows reconnecting and refetches after recovery |
| Duplicate stream event | Idempotent consumer produces one business effect |
| Slow socket client | Connection is bounded and disconnected, not unbounded memory |
| Stale write | Mutation rejected (412) |
| Conflicting schedule | Constraint rejects the invalid assignment |
| Bad tenant context | Database access fails closed |
| Full worker backlog | Queue age/lag metrics rise; system remains observable |

### Outbox crash points

| Case | Crash point | Required outcome |
|---|---|---|
| A | API crashes before commit | No business update, no outbox event |
| B | API crashes right after commit | Update and outbox event exist; relay publishes later |
| C | Relay publishes, crashes before marking published | Event may publish again; consumers dedupe |
| D | Worker executes, crashes before ACK | Stream redelivers; consumer detects prior processing; no duplicate effect |

### Asynchronous reliability checklist

- API can commit while the worker is stopped
- Event stays recoverable
- Worker restart processes the backlog
- A duplicate event does not duplicate the outcome
- A failed event retries
- A permanent failure enters the DLQ
- The backlog is observable

Delivery is **at-least-once with idempotent processing**. Nothing in the system claims exactly-once execution.

## 9. Metrics

```text
http_requests_total · http_request_duration_seconds · http_errors_total
db_pool_in_use · db_pool_wait_duration · db_query_duration
websocket_connections · websocket_subscriptions · websocket_reconnects · websocket_dropped_slow_clients
outbox_unpublished · outbox_oldest_event_seconds
worker_jobs_processed · worker_jobs_failed · worker_job_duration · worker_retry_count · dead_letter_count
redis_operation_duration
auth_failures · authorization_denials · tenant_rate_limit_hits
idempotency_replays_total · version_conflicts_total · schedule_conflicts_total · rls_context_missing_total
```

## 10. Performance method

Scenarios: **A** read-heavy work-order list · **B** mixed CRUD · **C** high-contention scheduling · **D** concurrent WebSocket clients · **E** realtime fan-out · **F** worker backlog drain · **G** duplicate command storm · **H** large report export.

Each published result includes throughput, p50/p95/p99, error rate, DB utilization, pool wait, Redis latency, queue lag, memory, CPU and WebSocket delivery latency, along with the hardware and workload definition.

Initial engineering **targets**. These are budgets to test against, not claims:

```text
ordinary CRUD API p95             < 300 ms under the defined load
local realtime propagation p95    < 500 ms
ordinary background queue age     < 5 s
invalid overlapping assignments     0
cross-tenant data disclosures       0
```

## 11. Gate checklists referenced by the plan

**Foundation exit:** API health endpoint · web app loads · PostgreSQL reachable · Redis reachable · Keycloak login round trip · migration system works · OTel collector receives a test trace · CI green · one documented command boots the system.

**Tenancy hostile statements:**
- A Cascade user cannot retrieve a Northstar record.
- A Cascade user cannot mutate a Northstar record.
- A Cascade DB context cannot SELECT a Northstar row.
- A Cascade row cannot FK-reference a Northstar entity.

## 12. Demo script (5–8 minutes)

1. Log in as a Cascade dispatcher.
2. Open a technician session in a second browser profile.
3. Assign a work order; the technician receives a live update.
4. Attempt an overlapping assignment; the database rejects it.
5. Open the same work order in two dispatcher sessions and make conflicting edits; the stale version is rejected.
6. Submit a duplicate command; idempotency prevents a duplicate operation.
7. Trigger an asynchronous report.
8. Kill the worker; the queue shows pending work.
9. Restart the worker; it recovers.
10. Open the trace and follow the operation from the API through the database, the event and the worker.
11. Switch to the Northstar tenant; its data is completely separate.
12. Show the automated tenant-isolation and concurrency results.

## 13. Deliberate non-goals (until a measured need exists)

- **Microservices.** The monolith keeps invariants transactional. Extraction paths are documented instead.
- **Kafka.** Redis Streams demonstrates durable consumer groups. An `EventBus` interface keeps a broker swap possible.
- **Elasticsearch.** PostgreSQL full-text search and `pg_trgm` come first.
- **Kubernetes.** Docker Compose is canonical (ADR-010).
- **Caching layer.** Added only after a benchmark shows PostgreSQL is insufficient.
- **AI assistant.** Optional, last, and only over controlled reporting data.
