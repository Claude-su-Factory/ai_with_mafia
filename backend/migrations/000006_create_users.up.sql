CREATE TABLE users (
    auth_id      TEXT PRIMARY KEY,
    player_id    TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
