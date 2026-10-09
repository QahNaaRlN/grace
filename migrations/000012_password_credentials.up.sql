CREATE TABLE auth_password_credentials (
 user_id UUID PRIMARY KEY REFERENCES users(id),
 password_hash TEXT NOT NULL,
 revision BIGINT NOT NULL DEFAULT 1,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE auth_password_tokens (
 token_hash CHAR(64) PRIMARY KEY,
 purpose TEXT NOT NULL CHECK(purpose IN ('register','reset')),
 user_id UUID NOT NULL REFERENCES users(id),
 email TEXT NOT NULL,
 invitation_hash CHAR(64),
 password_hash TEXT,
 revision BIGINT NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL,
 used_at TIMESTAMPTZ
);
CREATE INDEX auth_password_tokens_expiry ON auth_password_tokens(expires_at);
