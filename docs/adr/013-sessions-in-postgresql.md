# ADR-013: Server-side sessions stored in PostgreSQL

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

BFF sessions (ADR-006) need server storage. The failure matrix requires that a Redis outage degrades only async and realtime features, so sessions in Redis would log every user out during an outage.

## Decision

Store sessions in a non-tenant `sessions` table: a SHA-256 hash of a 256-bit random ID (the raw ID only ever lives in the cookie), the user, a 12h absolute expiry, a 2h sliding idle expiry, the refresh token encrypted with AES-256-GCM (key from env/secret file, with a key ID for rotation), and IP/user-agent metadata. The ID rotates at login. `last_seen_at` writes are throttled (at most one per minute).

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Redis sessions | Couples authentication to Redis availability |
| Stateless JWT cookie | Cannot revoke immediately; larger cookie |

## Consequences

- **Positive:** immediate revocation, consistent failure semantics.
- **Negative:** one indexed PK lookup per request.
- **Follow-ups:** expired-session cleanup job.

## Revisit if

Session lookup becomes a measured hot spot (then add a short in-process cache).
