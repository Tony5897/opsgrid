# ADR-014: Go standard library ServeMux for routing

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Since Go 1.22, `net/http.ServeMux` supports method matching and path wildcards (`r.PathValue`). The generated strict server needs only a standard `http.Handler`.

## Decision

Use `http.ServeMux`, with middleware as plain `func(http.Handler) http.Handler` composition in `platform/httpx`.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| chi / echo / gin | Extra dependency for features the stdlib now covers |

## Consequences

- **Positive:** zero routing dependencies.
- **Negative:** route groups are composed manually.

## Revisit if

Routing needs exceed what ServeMux provides (for example, regex constraints).
