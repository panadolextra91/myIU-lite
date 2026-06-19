CREATE TYPE user_role AS ENUM ('student', 'lecturer', 'admin');

CREATE TABLE users (
    id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username             TEXT NOT NULL,
    password_hash        TEXT NOT NULL,
    role                 user_role NOT NULL,
    must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ
);

CREATE UNIQUE INDEX users_username_active_uq ON users (username) WHERE deleted_at IS NULL;

CREATE TABLE audit_log (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id    BIGINT REFERENCES users(id),
    action      TEXT NOT NULL,
    target      TEXT,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_actor_idx   ON audit_log (actor_id);
CREATE INDEX audit_log_created_idx ON audit_log (created_at);
