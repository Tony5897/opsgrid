-- Tenancy: organizations, locations, memberships, counters (ADR-003, ADR-004, ADR-015).
-- Tenant tables ENABLE+FORCE RLS. organizations is the tenant root (policy on id).
-- organization_memberships allows InUserTx (own rows) OR InTenantTx (org scope).

-- +goose Up
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

CREATE TABLE app.organizations (
    id          uuid PRIMARY KEY DEFAULT uuidv7(),
    name        text NOT NULL,
    slug        citext NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_slug_key UNIQUE (slug)
);

CREATE TRIGGER organizations_touch_updated_at
    BEFORE UPDATE ON app.organizations
    FOR EACH ROW
    EXECUTE FUNCTION app.touch_updated_at();

ALTER TABLE app.organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.organizations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.organizations
    TO opsgrid_app, opsgrid_worker
    USING (id = (SELECT app.current_org_id()))
    WITH CHECK (id = (SELECT app.current_org_id()));

COMMENT ON TABLE app.organizations IS
    'Tenant root. Insert with InTenantTx scoped to the new org id (WITH CHECK on id).';

CREATE TABLE app.locations (
    id               uuid NOT NULL DEFAULT uuidv7(),
    organization_id  uuid NOT NULL REFERENCES app.organizations (id),
    name             text NOT NULL,
    timezone         text NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    CONSTRAINT locations_organization_id_id_key UNIQUE (organization_id, id)
);

CREATE TRIGGER locations_touch_updated_at
    BEFORE UPDATE ON app.locations
    FOR EACH ROW
    EXECUTE FUNCTION app.touch_updated_at();

ALTER TABLE app.locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.locations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.locations
    TO opsgrid_app, opsgrid_worker
    USING (organization_id = (SELECT app.current_org_id()))
    WITH CHECK (organization_id = (SELECT app.current_org_id()));

COMMENT ON TABLE app.locations IS
    'Org sites/branches; IANA timezone for schedule display.';

CREATE TABLE app.organization_memberships (
    organization_id  uuid NOT NULL REFERENCES app.organizations (id),
    user_id          uuid NOT NULL REFERENCES app.users (id),
    role             text NOT NULL,
    status           text NOT NULL DEFAULT 'active',
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, user_id),
    CONSTRAINT organization_memberships_role_check
        CHECK (role IN ('owner', 'dispatcher', 'technician', 'viewer')),
    CONSTRAINT organization_memberships_status_check
        CHECK (status IN ('active', 'removed'))
);

CREATE TRIGGER organization_memberships_touch_updated_at
    BEFORE UPDATE ON app.organization_memberships
    FOR EACH ROW
    EXECUTE FUNCTION app.touch_updated_at();

CREATE INDEX organization_memberships_user_id_idx
    ON app.organization_memberships (user_id);

ALTER TABLE app.organization_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.organization_memberships FORCE ROW LEVEL SECURITY;
CREATE POLICY membership_access ON app.organization_memberships
    TO opsgrid_app, opsgrid_worker
    USING (
        user_id = (SELECT app.current_user_id())
        OR organization_id = (SELECT app.current_org_id())
    )
    WITH CHECK (
        user_id = (SELECT app.current_user_id())
        OR organization_id = (SELECT app.current_org_id())
    );

COMMENT ON TABLE app.organization_memberships IS
    'User↔org roles. RLS: own rows (InUserTx) or org-scoped (InTenantTx).';

CREATE TABLE app.organization_counters (
    organization_id  uuid NOT NULL REFERENCES app.organizations (id),
    name             text NOT NULL,
    value            bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (organization_id, name),
    CONSTRAINT organization_counters_value_nonnegative CHECK (value >= 0)
);

ALTER TABLE app.organization_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.organization_counters FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.organization_counters
    TO opsgrid_app, opsgrid_worker
    USING (organization_id = (SELECT app.current_org_id()))
    WITH CHECK (organization_id = (SELECT app.current_org_id()));

COMMENT ON TABLE app.organization_counters IS
    'Per-org sequential counters (e.g. work_order numbers); UPDATE … RETURNING in the same tx (ADR-015).';

-- +goose Down
-- squawk-ignore ban-drop-table
DROP TABLE IF EXISTS app.organization_counters;
-- squawk-ignore ban-drop-table
DROP TABLE IF EXISTS app.organization_memberships;
-- squawk-ignore ban-drop-table
DROP TABLE IF EXISTS app.locations;
-- squawk-ignore ban-drop-table
DROP TABLE IF EXISTS app.organizations;
