# ADR-017: Local S3-compatible object storage server

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Attachments use direct browser uploads with presigned URLs. Local development must behave like Cloudflare R2: presigned PUT/GET, CORS for browser PUT, and content-length/type bound into the signature. MinIO's community distribution changed in 2025 (reduced console, source-only releases), which makes it a poor default.

## Decision

Use **Garage v2** (`dxflrs/garage`) as the local S3-compatible server. It runs as the `storage` Compose service and is provisioned idempotently by the one-shot `storage-init` job (`infra/garage/init.sh`) through Garage's admin API v2. The job creates the single-node layout, the app's access key, the `opsgrid` bucket, the grants, and a bucket CORS policy for the app origins. The application talks plain S3 (AWS SDK v2, path-style), so production and the public demo swap in Cloudflare R2 by configuration only.

### Spike results (2026-10-02)

Reproduce with `test/spikes/s3-compat/run.sh`. The spike used AWS SDK for Go v2 (`service/s3` v1.114.0) presigning, the same code path the G7 storage adapter will use.

| Criterion | Garage v2.4.1 | SeaweedFS 4.48 |
|---|---|---|
| `PutBucketCors` via S3 API | ✅ | ✅ |
| Browser CORS preflight allows the app origin | ✅ | ✅ |
| Preflight from a foreign origin is not allowed | ✅ | ✅ |
| Presigned PUT from the app origin (response carries `Access-Control-Allow-Origin`) | ✅ | ✅ |
| Signed `Content-Type` enforced (other type rejected) | ✅ | ✅ |
| Signed `Content-Length` enforced (oversize rejected) | ✅ | ✅ |
| Presigned GET works, then fails after expiry | ✅ | ✅ |
| `HEAD` reports size and type (finalize step) | ✅ | ✅ |
| Memory after the test (budget < 256 MB) | **35 MiB** | 65 MiB |

Both candidates pass every functional criterion. Garage is chosen because it uses about half the memory, which matters on 8 GB development machines that also run Keycloak and the observability profile, and because it is a single binary focused solely on S3. Its image has no shell, so provisioning runs through the admin HTTP API from a `curl` container. That adds one small init job, which is a deliberate trade for the smaller runtime. In the integrated stack Garage settles at about 29 MiB.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| SeaweedFS | Passed every criterion but uses about twice the memory; a viable fallback |
| MinIO | Distribution and maintenance changes |
| LocalStack | Heavier; S3 emulation fidelity varies |

## Consequences

- **Positive:** the local environment exercises the real direct-upload contract, including CORS and signed constraints, so G7 cannot pass locally and fail on R2 for protocol reasons.
- **Negative:** Garage's licence is AGPL-3.0. It is used only as an unmodified development service and is never linked into or distributed with OpsGrid, so the licence imposes no obligations on this project.
- **Follow-ups:** G7 storage adapter configuration (`S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`, credentials, path-style); an integration test against the `storage` service; the R2 bucket CORS policy mirrors `infra/garage/init.sh`.

## Revisit if

Garage fails a G7 acceptance test or becomes unmaintained; SeaweedFS is the measured fallback.
