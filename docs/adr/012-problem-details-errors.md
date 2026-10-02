# ADR-012: RFC 9457 Problem Details for all API errors

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Clients need stable, machine-readable error codes. Database errors must never reach the browser.

## Decision

Every non-2xx response is `application/problem+json` (RFC 9457) with `type`, `title`, `status` and `detail`, plus extension members `code` (stable enum: AUTHENTICATION_REQUIRED, FORBIDDEN, TENANT_NOT_FOUND, RESOURCE_NOT_FOUND, VALIDATION_FAILED, PRECONDITION_REQUIRED, VERSION_CONFLICT, SCHEDULE_CONFLICT, IDEMPOTENCY_CONFLICT, IDEMPOTENCY_KEY_REQUIRED, RATE_LIMITED, DEPENDENCY_UNAVAILABLE, INTERNAL_ERROR), `requestId`, `traceId` and optional `details`/`errors[]`. One function (`httpx.WriteError`) maps typed domain errors to problems; unknown errors become INTERNAL_ERROR with no detail.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Custom JSON error shape | Non-standard; tooling cannot recognize it |

## Consequences

- **Positive:** a standard media type; stable codes for the UI copy map.
- **Follow-ups:** fuzz/unit tests ensuring PG error text never appears in responses.

## Revisit if

Never; extensions can evolve compatibly.
