-- Identity: global users and server-side sessions (ADR-006, ADR-013).
-- Non-tenant tables: login bootstrap and session lookup by hash use
-- InSystemTx before any organization context exists. No RLS.

-- +goose Up
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

CREATE TABLE app.users (
    id            uuid PRIMARY KEY DEFAULT uuidv7(),
    issuer        text NOT NULL,
    subject       text NOT NULL,
    email         citext,
    display_name  text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_issuer_subject_key UNIQUE (issuer, subject)
);

CREATE TRIGGER users_touch_updated_at
    BEFORE UPDATE ON app.users
    FOR EACH ROW
    EXECUTE FUNCTION app.touch_updated_at();

COMMENT ON TABLE app.users IS
    'OIDC-linked identities; UNIQUE(issuer, subject). Global (no RLS); authorization is via memberships.';

CREATE TABLE app.sessions (
    id_hash              bytea PRIMARY KEY,
    user_id              uuid NOT NULL REFERENCES app.users (id) ON DELETE CASCADE,
    created_at           timestamptz NOT NULL DEFAULT now(),
    last_seen_at         timestamptz NOT NULL DEFAULT now(),
    absolute_expires_at  timestamptz NOT NULL,
    idle_expires_at      timestamptz NOT NULL,
    refresh_token_enc    bytea NOT NULL,
    refresh_token_key_id text NOT NULL,
    ip                   inet,
    user_agent           text,
    CONSTRAINT sessions_id_hash_len CHECK (octet_length(id_hash) = 32)
);

CREATE INDEX sessions_user_id_idx ON app.sessions (user_id);
CREATE INDEX sessions_absolute_expires_at_idx ON app.sessions (absolute_expires_at);

COMMENT ON TABLE app.sessions IS
    'BFF sessions keyed by SHA-256 of the cookie id (ADR-013). Raw id never stored. No RLS.';
COMMENT ON COLUMN app.sessions.id_hash IS
    'SHA-256 digest of the session secret; cookie holds only the raw id.';
COMMENT ON COLUMN app.sessions.refresh_token_enc IS
    'AES-256-GCM sealed IdP refresh token; key id selects the envelope key.';

-- +goose Down
-- squawk-ignore ban-drop-table
DROP TABLE IF EXISTS app.sessions;
-- squawk-ignore ban-drop-table
DROP TABLE IF EXISTS app.users;
