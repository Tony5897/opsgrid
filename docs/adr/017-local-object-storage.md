# ADR-017: Local S3-compatible object storage server

- **Status:** Proposed
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Attachments use direct browser uploads with presigned URLs. Local development must behave like Cloudflare R2: presigned PUT/GET, CORS for browser PUT, and content-length/type bound into the signature. MinIO's community distribution changed in 2025 (reduced console, source-only releases), which makes it a poor default.

## Decision

**Pending a spike (Gate 0, needs Docker).** Candidates: **Garage** (`dxflrs/garage`, v2.x) and **SeaweedFS** (S3 gateway). Acceptance: a presigned PUT from a browser origin succeeds with CORS; a presigned GET expires; a wrong content-type or oversize upload is rejected; it runs in under 256 MB of RAM. Record the measurements here.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| MinIO | Distribution and maintenance changes |
| LocalStack | Heavier; S3 emulation fidelity varies |

## Consequences

- **Follow-ups:** the storage interface is unchanged regardless of the server.

## Revisit if

Either candidate fails an acceptance criterion.
