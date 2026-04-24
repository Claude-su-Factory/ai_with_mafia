CREATE TABLE game_metrics (
    game_id                 TEXT PRIMARY KEY,
    room_id                 TEXT NOT NULL,
    started_at              TIMESTAMPTZ NOT NULL,
    ended_at                TIMESTAMPTZ,
    humans_count            INT  NOT NULL DEFAULT 0,
    ai_count                INT  NOT NULL DEFAULT 0,
    rounds                  INT,
    winner                  TEXT,
    tokens_in               BIGINT NOT NULL DEFAULT 0,
    tokens_out              BIGINT NOT NULL DEFAULT 0,
    cache_read_tokens       BIGINT NOT NULL DEFAULT 0,
    cache_creation_tokens   BIGINT NOT NULL DEFAULT 0,
    ad_impressions_lobby    INT  NOT NULL DEFAULT 0,
    ad_impressions_waiting  INT  NOT NULL DEFAULT 0,
    ad_impressions_result   INT  NOT NULL DEFAULT 0,
    quick_match_joins       INT  NOT NULL DEFAULT 0,
    quick_match_creates     INT  NOT NULL DEFAULT 0,
    quick_match_latency_ms  INT,
    truncated_turns         INT  NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_game_metrics_started_at ON game_metrics(started_at);
CREATE INDEX idx_game_metrics_room_id    ON game_metrics(room_id);
