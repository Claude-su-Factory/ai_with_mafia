CREATE TABLE IF NOT EXISTS game_results (
    id           TEXT PRIMARY KEY,
    room_id      TEXT NOT NULL,
    winner_team  TEXT NOT NULL,
    round_count  INT  NOT NULL DEFAULT 1,
    duration_sec INT  NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS game_result_players (
    id             TEXT PRIMARY KEY,
    game_result_id TEXT NOT NULL,
    player_id      TEXT NOT NULL,
    player_name    TEXT NOT NULL,
    role           TEXT NOT NULL,
    is_ai          BOOLEAN NOT NULL DEFAULT FALSE,
    survived       BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_game_results_room ON game_results(room_id);
