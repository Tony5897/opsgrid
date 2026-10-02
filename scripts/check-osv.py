#!/usr/bin/env python3
"""Fails if osv-scanner's JSON output contains any finding whose ID is not
explicitly documented as accepted below. Used instead of osv-scanner's own
exit code: that code was observed to be inconsistent across otherwise
identical runs of v2.2.3 (sometimes 0, 1 or 127 for the same findings),
so CI makes its own pass/fail decision from the structured results.
"""
import json
import sys

# id -> one-line reason it is not actionable here. Keep in sync with any
# `-- +goose` style audit trail in commit messages when an entry is added.
ACCEPTED = {
    "GO-2026-5932": (
        "golang.org/x/crypto/openpgp: unmaintained/unsafe by design, no "
        "fixed version exists (blanket advisory). We never import openpgp; "
        "x/crypto reaches us only via testcontainers-go's ssh client, used "
        "solely by internal/platform/database/dbtest under the "
        "`integration` build tag (never compiled into a shipped binary)."
    ),
}


def main() -> int:
    path = sys.argv[1] if len(sys.argv) > 1 else "/dev/stdin"
    with open(path, encoding="utf-8") as f:
        data = json.load(f)

    unaccepted: dict[str, list[str]] = {}
    for result in data.get("results", []):
        source = result.get("source", {}).get("path", "?")
        for pkg in result.get("packages", []):
            name = pkg.get("package", {}).get("name", "?")
            for vuln in pkg.get("vulnerabilities", []):
                vid = vuln["id"]
                if vid in ACCEPTED:
                    print(f"accepted: {vid} ({name}, {source}) — {ACCEPTED[vid]}")
                    continue
                unaccepted.setdefault(vid, []).append(f"{name} ({source})")

    if unaccepted:
        print("\nUnaccepted vulnerabilities found:", file=sys.stderr)
        for vid, locations in sorted(unaccepted.items()):
            print(f"  {vid}: {', '.join(locations)}", file=sys.stderr)
        print(
            "\nFix the package, or add the ID to ACCEPTED in "
            "scripts/check-osv.py with a reason.",
            file=sys.stderr,
        )
        return 1

    print("\nNo unaccepted vulnerabilities.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
