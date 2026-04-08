CREATE TABLE IF NOT EXISTS game_states (
    room_id     TEXT PRIMARY KEY,
    phase       TEXT        NOT NULL,
    round       INT         NOT NULL DEFAULT 1,
    players_json JSONB      NOT NULL DEFAULT '[]',
    night_kills  JSONB      NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
