# ADR-018: Logs: slog JSON on stdout; shipping decided by OTel log maturity

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Logs must be structured and correlated with traces. The OpenTelemetry Go logs signal has trailed traces and metrics in stability.

## Decision

Every process writes `log/slog` JSON to stdout with `trace_id`, `span_id`, `request_id`, `organization_id` and `actor_user_id`, after a redaction `ReplaceAttr`. stdout is the source of truth. For Loki: at Gate 8, if the OTel Go logs SDK and `otelslog` bridge are stable, export via the Collector's OTLP logs pipeline; otherwise use Grafana Alloy's Docker log discovery. Record the outcome here.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Promtail | Deprecated in favor of Alloy |

## Consequences

- **Positive:** logs are useful even without a shipping pipeline.

## Revisit if

The OTel Go logs signal reaches stable (then standardize on OTLP).
