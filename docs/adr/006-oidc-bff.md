# ADR-006: OpenID Connect via a backend-for-frontend session

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Authentication must use a real identity protocol. Storing access or refresh tokens in browser JavaScript exposes them to XSS. Implicit and password grants are disallowed by current OAuth security guidance (RFC 9700).

## Decision

Keycloak is the development IdP. The Go API acts as a **confidential OIDC client** using the Authorization Code flow **with PKCE (S256)**, `state` and `nonce`. After the callback it creates a server-side session (ADR-013) and sets a `__Host-opsgrid_session` cookie (Secure, HttpOnly, SameSite=Lax, Path=/). The SPA never sees IdP tokens. Keycloak answers *who authenticated*; PostgreSQL memberships answer *what they may do in which organization*. Logout is RP-initiated, plus back-channel logout.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| SPA public client with tokens in memory | Tokens are reachable by XSS; refresh-token handling in the browser |
| Home-grown password login | Reimplements security-critical code |

## Consequences

- **Positive:** no tokens in JS; standards-based; the IdP can be swapped.
- **Negative:** the API holds session state; requires CSRF defenses (ADR-019).
- **Follow-ups:** realm committed as JSON; login round-trip E2E test.

## Revisit if

The product needs third-party API clients that must hold their own tokens, which would add a token-based API path alongside the BFF.
