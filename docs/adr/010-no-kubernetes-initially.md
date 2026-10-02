# ADR-010: Docker Compose, not Kubernetes, as the canonical runtime

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

The project needs a reproducible multi-service environment. Kubernetes adds cluster operations that do not improve domain correctness.

## Decision

Docker Compose (with profiles: default, `observability`, `chaos`) is the canonical development, CI and demo runtime. Images are deployment-neutral (distroless, 12-factor config), so orchestration can be added later.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Kubernetes + Helm from day one | Complexity without a present need |
| Bare processes | Not reproducible |

## Consequences

- **Positive:** `git clone && make dev`; laptop-friendly.
- **Negative:** no autoscaling or rolling-deployment demonstration initially.
- **Follow-ups:** a Kubernetes/Helm stretch goal after v1.0.

## Revisit if

The number of services grows materially, operational requirements demand orchestration, or Kubernetes itself becomes a portfolio objective.
