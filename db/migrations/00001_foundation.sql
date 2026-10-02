-- Foundation: extensions, the app schema, privileges and the tenant-context
-- helper functions every RLS policy uses (ADR-004).
--
-- Runs as opsgrid_migrator (database owner). Roles come from
-- db/bootstrap/roles.sql.

-- +goose Up

-- Trusted extensions: creatable by the database owner.
CREATE EXTENSION IF NOT EXISTS btree_gist; -- scalar = alongside range && in exclusion constraints
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram search
CREATE EXTENSION IF NOT EXISTS citext;     -- case-insensitive email

-- Nobody but the owner may create objects in public; runtime roles may only
-- read the goose version table (for readiness checks).
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO opsgrid_app, opsgrid_worker, opsgrid_relay;
GRANT SELECT ON public.goose_db_version TO opsgrid_app, opsgrid_worker, opsgrid_relay;

CREATE SCHEMA app;
REVOKE ALL ON SCHEMA app FROM PUBLIC;
GRANT USAGE ON SCHEMA app TO opsgrid_app, opsgrid_worker, opsgrid_relay, opsgrid_reporting_ro;

-- Default privileges for objects the migrator creates in schema app.
-- DML for runtime roles; tables that must be append-only (audit) or
-- restricted (outbox) REVOKE explicitly in their own migrations.
ALTER DEFAULT PRIVILEGES FOR ROLE opsgrid_migrator IN SCHEMA app
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO opsgrid_app, opsgrid_worker;
ALTER DEFAULT PRIVILEGES FOR ROLE opsgrid_migrator IN SCHEMA app
    GRANT SELECT ON TABLES TO opsgrid_reporting_ro;
ALTER DEFAULT PRIVILEGES FOR ROLE opsgrid_migrator IN SCHEMA app
    GRANT USAGE, SELECT ON SEQUENCES TO opsgrid_app, opsgrid_worker;
ALTER DEFAULT PRIVILEGES FOR ROLE opsgrid_migrator IN SCHEMA app
    REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE opsgrid_migrator IN SCHEMA app
    GRANT EXECUTE ON FUNCTIONS TO opsgrid_app, opsgrid_worker, opsgrid_relay, opsgrid_reporting_ro;

-- Tenant context accessors.
--
-- current_setting(..., true) returns NULL when the setting was never defined
-- in this session, but '' (empty string) once a transaction-local
-- set_config() has been used and that transaction ended. Pooled connections
-- hit the second case constantly, and ''::uuid raises an error. NULLIF makes
-- both cases NULL, and "organization_id = NULL" is never true, so a missing
-- context fails closed: zero rows visible, every write rejected.
--
-- Policies must call these as (SELECT app.current_org_id()) so the planner
-- evaluates them once per statement (InitPlan) rather than once per row.
CREATE FUNCTION app.current_org_id() RETURNS uuid
    LANGUAGE sql STABLE PARALLEL SAFE
    AS $$ SELECT NULLIF(current_setting('app.organization_id', true), '')::uuid $$;

CREATE FUNCTION app.current_user_id() RETURNS uuid
    LANGUAGE sql STABLE PARALLEL SAFE
    AS $$ SELECT NULLIF(current_setting('app.user_id', true), '')::uuid $$;

COMMENT ON FUNCTION app.current_org_id() IS
    'Transaction-local tenant context set by platform/database.InTenantTx; NULL (fail-closed) when unset.';
COMMENT ON FUNCTION app.current_user_id() IS
    'Transaction-local acting user set by platform/database.InTenantTx/InUserTx; NULL when unset.';

-- Shared trigger: maintain updated_at on mutable tables.
CREATE FUNCTION app.touch_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END
$$;

-- Shared trigger: make a table append-only regardless of grants (audit).
CREATE FUNCTION app.reject_mutation() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION 'table %.% is append-only', TG_TABLE_SCHEMA, TG_TABLE_NAME
        USING ERRCODE = 'insufficient_privilege';
END
$$;

-- +goose Down
DROP FUNCTION IF EXISTS app.reject_mutation();
DROP FUNCTION IF EXISTS app.touch_updated_at();
DROP FUNCTION IF EXISTS app.current_user_id();
DROP FUNCTION IF EXISTS app.current_org_id();
DROP SCHEMA IF EXISTS app;
REVOKE SELECT ON public.goose_db_version FROM opsgrid_app, opsgrid_worker, opsgrid_relay;
REVOKE USAGE ON SCHEMA public FROM opsgrid_app, opsgrid_worker, opsgrid_relay;
