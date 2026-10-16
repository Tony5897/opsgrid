-- Expand membership roles to include admin (architecture overview §6).
-- Expanding the allowlist; not weakening isolation. Table is empty/tiny at G1;
-- goose wraps Up/Down in one transaction so NOT VALID + VALIDATE cannot be split.

-- +goose Up
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

ALTER TABLE app.organization_memberships
    DROP CONSTRAINT organization_memberships_role_check;

-- squawk-ignore constraint-missing-not-valid
ALTER TABLE app.organization_memberships ADD CONSTRAINT organization_memberships_role_check CHECK (role IN ('owner', 'admin', 'dispatcher', 'technician', 'viewer'));

-- +goose Down
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

-- Reject if any admin rows exist before shrinking the allowlist.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM app.organization_memberships WHERE role = 'admin'
    ) THEN
        RAISE EXCEPTION 'cannot downgrade: organization_memberships still has role=admin rows';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE app.organization_memberships
    DROP CONSTRAINT organization_memberships_role_check;

-- squawk-ignore constraint-missing-not-valid
ALTER TABLE app.organization_memberships ADD CONSTRAINT organization_memberships_role_check CHECK (role IN ('owner', 'dispatcher', 'technician', 'viewer'));
