-- +goose Up
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
                                     id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
                                     email citext NOT NULL UNIQUE,
                                     password_hash text NOT NULL,
                                     created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
                                              id uuid PRIMARY KEY,
                                              user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                              token_hash text NOT NULL UNIQUE,
                                              expires_at timestamptz NOT NULL,
                                              revoked_at timestamptz NULL,
                                              created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
COMMIT;
