#!/usr/bin/env bash
# Runs once, on first container start with an empty data directory.
# Applies the environment-neutral role bootstrap, sets role passwords from
# the environment, and creates the application database.
set -euo pipefail

psql_su() { psql -v ON_ERROR_STOP=1 --no-psqlrc --username "$POSTGRES_USER" --dbname postgres "$@"; }

echo "opsgrid: bootstrapping roles"
psql_su -f /opsgrid-bootstrap/roles.sql

for role in migrator app worker relay reporting_ro; do
  var="OPSGRID_$(echo "$role" | tr '[:lower:]' '[:upper:]')_PASSWORD"
  pw="${!var:-}"
  if [[ -z "$pw" ]]; then
    echo "opsgrid: $var is not set" >&2
    exit 1
  fi
  # psql variables are interpolated from stdin (not -c), and :'pw' quotes safely.
  psql_su -v pw="$pw" <<<"ALTER ROLE opsgrid_${role} PASSWORD :'pw';"
done

echo "opsgrid: creating database ${OPSGRID_DB:-opsgrid}"
psql_su -v dbname="${OPSGRID_DB:-opsgrid}" -f /opsgrid-bootstrap/database.sql
