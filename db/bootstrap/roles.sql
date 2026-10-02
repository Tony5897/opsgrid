-- OpsGrid database roles (cluster-level; run once per cluster as a superuser).
--
-- Roles are created WITHOUT passwords; the environment-specific bootstrap
-- (infra/postgres/initdb, test harness, or cloud provisioning) sets them.
-- Idempotent: safe to re-run.
--
--   opsgrid_migrator      owns the database and every object in schema app
--   opsgrid_app           API runtime: DML only, subject to RLS
--   opsgrid_worker        worker runtime: DML only, subject to RLS
--   opsgrid_relay         outbox relay: reads/updates outbox_events only
--   opsgrid_reporting_ro  read-only reporting access, subject to RLS
--
-- No runtime role is a superuser, owns tables, or has BYPASSRLS (ADR-004).

DO $$
DECLARE
    r text;
BEGIN
    FOREACH r IN ARRAY ARRAY[
        'opsgrid_migrator', 'opsgrid_app', 'opsgrid_worker',
        'opsgrid_relay', 'opsgrid_reporting_ro'
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = r) THEN
            EXECUTE format(
                'CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS',
                r
            );
        END IF;
    END LOOP;
END
$$;

-- Defensive: re-assert attributes in case a role pre-existed with more power.
ALTER ROLE opsgrid_migrator     NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
ALTER ROLE opsgrid_app          NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
ALTER ROLE opsgrid_worker       NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
ALTER ROLE opsgrid_relay        NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
ALTER ROLE opsgrid_reporting_ro NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;

-- Runtime guard rails. A runaway query or an abandoned transaction must not
-- hold locks indefinitely.
ALTER ROLE opsgrid_app    SET statement_timeout = '5s';
ALTER ROLE opsgrid_app    SET idle_in_transaction_session_timeout = '10s';
ALTER ROLE opsgrid_app    SET lock_timeout = '3s';
ALTER ROLE opsgrid_worker SET statement_timeout = '60s';
ALTER ROLE opsgrid_worker SET idle_in_transaction_session_timeout = '30s';
ALTER ROLE opsgrid_worker SET lock_timeout = '5s';
ALTER ROLE opsgrid_relay  SET statement_timeout = '5s';
ALTER ROLE opsgrid_relay  SET idle_in_transaction_session_timeout = '10s';
ALTER ROLE opsgrid_reporting_ro SET default_transaction_read_only = on;
ALTER ROLE opsgrid_reporting_ro SET statement_timeout = '30s';
