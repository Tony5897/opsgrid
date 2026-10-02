# ADR-019: CSRF defense in depth

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

A cookie-based session (ADR-006) needs protection against cross-site requests.

## Decision

Three layers: (1) the `__Host-` session cookie with `SameSite=Lax`; (2) Go's `http.CrossOriginProtection` (Fetch Metadata `Sec-Fetch-Site` / `Origin` checks, Go ≥ 1.25) on all unsafe methods; (3) a required custom header `X-OpsGrid-CSRF: 1` on unsafe methods, which cross-site forms cannot set without a CORS preflight that the API never grants. The WebSocket handshake validates `Origin` against an allowlist.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Synchronizer tokens | More plumbing for equal protection given same-origin serving (ADR-020) |

## Consequences

- **Follow-ups:** tests for cross-site POST rejection and a missing header.

## Revisit if

The API must be called cross-origin by browsers.
