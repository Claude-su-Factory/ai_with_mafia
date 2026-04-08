CREATE TABLE IF NOT EXISTS ai_histories (
    room_id      TEXT        NOT NULL,
    player_id    TEXT        NOT NULL,
    history_json JSONB       NOT NULL DEFAULT '[]',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, player_id)
);
