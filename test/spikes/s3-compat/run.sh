#!/usr/bin/env bash
# Re-runs the ADR-017 object-storage spike against Garage and SeaweedFS.
set -euo pipefail
cd "$(dirname "$0")"
root=$(cd ../../.. && pwd)
ACCESS=GK0123456789abcdef01234567
SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

docker compose up -d --wait >/dev/null 2>&1 || docker compose up -d >/dev/null
trap 'docker compose down -v >/dev/null 2>&1' EXIT

# Garage: single-node layout, imported key, bucket and grants.
for _ in $(seq 1 30); do docker compose exec -T garage /garage status >/dev/null 2>&1 && break; sleep 1; done
node=$(docker compose exec -T garage /garage status 2>/dev/null | awk '/^[0-9a-f]{16}/{print $1; exit}')
docker compose exec -T garage /garage layout assign -z dc1 -c 1G "$node" >/dev/null
docker compose exec -T garage /garage layout apply --version 1 >/dev/null
docker compose exec -T garage /garage key import --yes -n spike "$ACCESS" "$SECRET" >/dev/null
docker compose exec -T garage /garage bucket create spike >/dev/null
docker compose exec -T garage /garage bucket allow --read --write --owner spike --key spike >/dev/null

# SeaweedFS: wait for the S3 gateway.
for _ in $(seq 1 60); do curl -s -o /dev/null localhost:18333 && break; sleep 1; done

cd "$root"
GOTOOLCHAIN=go1.27.1 S3_SPIKE_ACCESS=$ACCESS S3_SPIKE_SECRET=$SECRET \
  go test -tags spike -count=1 -v ./test/spikes/s3-compat/ 2>&1 | grep -E '^(=== RUN|--- |\s+spike_test|PASS|FAIL|ok)' || true

echo
echo "Memory after test:"
docker stats --no-stream --format '  {{.Name}}: {{.MemUsage}}' $(docker compose -f test/spikes/s3-compat/compose.yaml ps -q)
