# ADR-022: Humans commit, automation reports

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Dependency bots (Renovate, Dependabot version updates) and release bots (release-please) open pull requests and create commits under bot identities. This repository treats every commit as a reviewed, attributable engineering decision. Its history is also part of the project's evidence, so it should show who decided each change and why.

## Decision

No automation creates commits, branches or pull requests in this repository.

- Automation **reports**: CI checks, CodeQL and Scorecard findings, and a weekly *Dependency report* issue (`.github/workflows/deps-report.yml`, `make deps`) that lists available upgrades for Go modules, Go tools, pnpm packages and container images.
- The maintainer **applies** upgrades: update the pin, run the full check suite, and commit with a Conventional Commit message (`chore(deps): ...`).
- Releases and changelogs are cut manually from Conventional Commit history.
- Supply-chain protections stay automated because they do not commit: pnpm `minimumReleaseAge` (24 h), SHA-pinned actions, OSV-Scanner, govulncheck, Trivy, signed images with an SBOM and provenance.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Renovate / Dependabot PRs | Bot-authored commits; upgrade decisions become rubber stamps |
| release-please | Bot-authored release commits and PRs |

## Consequences

- **Positive:** a fully attributable history; every upgrade is consciously reviewed and tested.
- **Negative:** upgrades take deliberate time, and the weekly report has to be acted on.
- **Follow-ups:** security advisories (GitHub alerts, OSV, govulncheck) are triaged within a week; critical ones immediately.

## Revisit if

The number of dependencies makes manual upgrades a measurable source of lag on security fixes.
