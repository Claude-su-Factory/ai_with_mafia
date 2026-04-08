CREATE TABLE IF NOT EXISTS rooms (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    game_type   TEXT NOT NULL,
    visibility  TEXT NOT NULL DEFAULT 'public',
    join_code   TEXT,
    host_id     TEXT NOT NULL,
    max_humans  INT  NOT NULL DEFAULT 1,
    status      TEXT NOT NULL DEFAULT 'waiting',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rooms_visibility ON rooms(visibility);
CREATE INDEX IF NOT EXISTS idx_rooms_join_code  ON rooms(join_code) WHERE join_code IS NOT NULL;
