-- Auth schema: users, credentials (WebAuthn), sessions, invitations.
-- Roles: 'user' | 'admin'. Admin can invite users.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    role        VARCHAR(16)  NOT NULL DEFAULT 'user',
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    invited_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_role_check CHECK (role IN ('user', 'admin')),
    CONSTRAINT users_status_check CHECK (status IN ('active', 'suspended'))
);

CREATE INDEX idx_users_email ON users(email);

CREATE TABLE credentials (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id    BYTEA NOT NULL UNIQUE,
    public_key       BYTEA NOT NULL,
    attestation_type VARCHAR(32),
    aaguid           BYTEA,
    sign_count       BIGINT NOT NULL DEFAULT 0,
    clone_warning    BOOLEAN DEFAULT FALSE,
    backup_eligible  BOOLEAN DEFAULT FALSE,
    transports       JSONB,
    device_name      VARCHAR(100),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at     TIMESTAMPTZ
);

CREATE INDEX idx_credentials_user_id ON credentials(user_id);

CREATE TABLE sessions (
    id               VARCHAR(64) PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address       VARCHAR(45),
    user_agent       TEXT,
    device_name      VARCHAR(100),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at       TIMESTAMPTZ NOT NULL,
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at       TIMESTAMPTZ,
    revoked_by       UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at) WHERE revoked_at IS NULL;

CREATE TABLE invitations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL,
    token         VARCHAR(64) NOT NULL UNIQUE,
    default_role  VARCHAR(16) NOT NULL DEFAULT 'user',
    invited_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL,
    used_at       TIMESTAMPTZ,
    used_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    is_bootstrap  BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT invitations_role_check CHECK (default_role IN ('user', 'admin')),
    CONSTRAINT invitations_status_check CHECK (status IN ('pending', 'used', 'expired', 'revoked'))
);

CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_email ON invitations(email);
CREATE INDEX idx_invitations_pending ON invitations(status) WHERE status = 'pending';
