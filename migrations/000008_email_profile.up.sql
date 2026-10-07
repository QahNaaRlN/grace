ALTER TABLE users ADD COLUMN first_name TEXT NOT NULL DEFAULT '' CHECK (length(first_name) <= 80);
ALTER TABLE users ADD COLUMN last_name TEXT NOT NULL DEFAULT '' CHECK (length(last_name) <= 80);
ALTER TABLE users ADD COLUMN avatar_png BYTEA CHECK (octet_length(avatar_png) <= 400000);
ALTER TABLE auth_invitations ADD COLUMN bound_email TEXT;
CREATE TABLE auth_email_challenges (
    browser_hash CHAR(64) PRIMARY KEY,
    code_hash CHAR(64) NOT NULL,
    email TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('login','link')),
    user_id UUID REFERENCES users(id),
    invitation_hash TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5)
);
CREATE INDEX auth_email_challenges_expiry_idx ON auth_email_challenges(expires_at);
CREATE TABLE auth_email_rate_windows (
    key_hash CHAR(64) PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    requests INTEGER NOT NULL CHECK (requests > 0)
);
CREATE INDEX auth_email_rate_expiry_idx ON auth_email_rate_windows(expires_at);
