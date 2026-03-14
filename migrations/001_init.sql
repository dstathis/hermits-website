CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    format      TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    date        TIMESTAMPTZ NOT NULL,
    location    TEXT NOT NULL DEFAULT '',
    location_url TEXT NOT NULL DEFAULT '',
    entry_fee   TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE admin_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscribers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL DEFAULT '',
    token      TEXT NOT NULL DEFAULT encode(gen_random_bytes(32), 'hex'),
    confirmed  BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_events_date ON events(date);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
CREATE INDEX idx_subscribers_token ON subscribers(token);
