# ADR-011: OpenAPI contract version

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

The API contract (`api/openapi.yaml`) generates the Go server interfaces and the TypeScript client. OpenAPI 3.2.1 is the newest specification, but generator support may lag behind it.

## Decision

Author the contract in **OpenAPI 3.1.1**: the highest version that both generators support fully, as measured by the Gate 0 spike below. Revisit 3.2.x when oapi-codegen supports it (see *Revisit if*).

### Spike results (2026-10-02)

The sample contract exercised path, query (array, `form`/`explode`) and header parameters (`Idempotency-Key`, `If-Match`), an `ETag` response header, `application/problem+json` default responses, cursor pagination, `additionalProperties: false`, and a `oneOf` with a `discriminator` mapping. A second variant added 3.1-style nullability (`type: [string, "null"]`).

| Generator | 3.0.4 | 3.1.1 | 3.1.1 + `type: [T, "null"]` | 3.2.1 | 3.2.1 + `type: [T, "null"]` |
|---|---|---|---|---|---|
| oapi-codegen v2.8.0 (strict std-http server + models) | ✅ compiles | ✅ compiles | ✅ `*string`, `*openapi_types.UUID` | ✅ only with 3.0-compatible syntax (output byte-identical to 3.0.4) | ❌ **fails**: "error generating Go schema for property 'assignedTo'" |
| @hey-api/openapi-ts v0.99.0 (types) | ✅ | ✅ | ✅ `string \| null` | ✅ | ✅ |

Findings:

- oapi-codegen accepts a 3.2.1 header but processes the document with 3.0 semantics. The first 3.1/3.2 schema feature breaks generation, so "3.2.1 works" would have been a false positive.
- hey-api generates a correctly tagged TypeScript union from the discriminator mapping.
- Generated Go compiled cleanly (`go vet`) for every version that generated.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Always newest | Risks broken generation |
| Code-first spec generation | The contract would follow the code instead of leading it |

## Consequences

- **Positive:** both server and client are generated from one contract with no hand patches, and 3.1 brings JSON Schema 2020-12 semantics (type arrays, `const`, `examples`).
- **Negative:** 3.2 features (for example the `QUERY` method and streaming media types) are unavailable for now. None are needed for the planned API.
- **Follow-ups:** `api/openapi.yaml` declares `openapi: 3.1.1`. CI regenerates both clients and fails on a diff. Redocly lints the bundle and oasdiff blocks unflagged breaking changes.

## Revisit if

oapi-codegen generates the 3.2.1 + `type: [T, "null"]` sample successfully. Re-run the spike script when Renovate bumps oapi-codegen.
