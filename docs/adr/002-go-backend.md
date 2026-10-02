# ADR-002: Go for the backend

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

The backend must demonstrate typed service engineering, cancellation, concurrency (workers, WebSocket hub, relay), graceful shutdown and race-free code. The frontend stays TypeScript.

## Decision

Use Go 1.27.x (pinned via the `go`/`toolchain` directives in `go.mod`). Standard library first: `net/http`, `log/slog`, `context`, `testing` (including `testing/synctest` and fuzzing). Third-party libraries only where the stdlib has no equivalent (pgx, go-redis, coder/websocket, go-oidc, OpenTelemetry).

## Alternatives considered

| Option | Why not chosen |
|---|---|
| TypeScript/Node backend | Same language as the frontend; adds no new systems evidence; weaker concurrency story |
| Rust | Higher learning cost per feature; slower iteration for a single engineer |
| Java/Kotlin + Spring | Heavier runtime and framework magic that obscures the mechanisms being demonstrated |

## Consequences

- **Positive:** static binaries, distroless images, a built-in race detector and fuzzer, and context propagation everywhere.
- **Negative:** more boilerplate than framework-heavy stacks; generics are less expressive.
- **Follow-ups:** golangci-lint v2 config; `go test -race` in CI.

## Revisit if

Go's toolchain or ecosystem blocks a core requirement (for example, an OIDC or OTel library becomes unmaintained with no alternative).
