# AI Mafia Game Platform

AI와 사람이 함께 플레이하는 마피아 게임 플랫폼.

- **Backend**: Go + Fiber v2 · WebSocket · pgx/v5 · go-redis/v9 · Anthropic Claude SDK
- **Frontend**: React 18 + TypeScript + Vite + Zustand
- **Auth**: Supabase (Google OAuth, ES256 JWT via JWK)
- **Infra (로컬)**: docker-compose 으로 Postgres 16 + Redis 7

자세한 내용은 [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) · [`docs/STATUS.md`](docs/STATUS.md) · [`docs/ROADMAP.md`](docs/ROADMAP.md) 참고.

---

## 빠른 시작 (로컬)

```bash
# 1. Postgres + Redis 기동
docker compose up -d

# 2. DB 스키마 생성 — 아래 "DB Schema" 섹션의 DDL 을 psql 에 붙여넣어 직접 실행
psql "postgres://postgres:password@localhost:5432/ai_playground"
# (DDL 복사·붙여넣기, 또는 README의 DDL 만 따로 .sql 파일로 저장 후 \i schema.sql)

# 3. backend 설정 파일 준비
cp backend/config.example.toml backend/config.toml
# config.toml 의 api_key, supabase JWK 값을 실제 값으로 채움

# 4. backend 실행
cd backend && go run ./cmd/server

# 5. frontend 실행 (다른 터미널)
cp frontend/.env.example frontend/.env.development
# .env.development 의 VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY 입력
cd frontend && npm install && npm run dev
```

---

## DB Schema

> **정책 (MANDATORY)**
> - **자동 마이그레이션 도구를 사용하지 않는다.** 이전에 사용하던 `golang-migrate` 는 제거됨. 서버 부팅 시 스키마를 건드리지 않는다
> - **스키마 변경은 항상 사람이 직접 적용한다** — 이 README 의 DDL 을 갱신하고, psql 등으로 수동 실행
> - **외래 키(FOREIGN KEY) 사용 금지** — 성능과 분산 환경(여러 Pod 가 동일 DB 에 동시 INSERT 하는 상황)에서의 복잡도 회피. 무결성은 애플리케이션 레이어에서 책임진다

아래 DDL 을 한 번 실행하면 전체 스키마가 만들어진다. 모든 `CREATE TABLE` 은 `IF NOT EXISTS` 가드라 재실행 안전.

### 1) `rooms` — 방 메타데이터

```sql
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
```

`RoomService` 가 인메모리 primary 로 운영하며 이 테이블은 recovery·재시작 시 복원 용도. 자세한 결정 배경: ARCHITECTURE §4.3.

### 2) `game_results` + `game_result_players` — 게임 결과 이력

```sql
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
```

`game_result_players.game_result_id` 는 `game_results.id` 를 참조하지만 **FK 는 걸지 않는다** (정책). 무결성은 `GameResultRepository.Save` 트랜잭션이 책임. T21 이후 `game_results.id` ↔ `game_metrics.game_id` 동일 UUID 사용.

### 3) `game_states` — 진행 중 게임 체크포인트

```sql
CREATE TABLE IF NOT EXISTS game_states (
    room_id      TEXT PRIMARY KEY,
    phase        TEXT        NOT NULL,
    round        INT         NOT NULL DEFAULT 1,
    players_json JSONB       NOT NULL DEFAULT '[]',
    night_kills  JSONB       NOT NULL DEFAULT '{}',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

LeaderLock 이 보유한 Pod 가 매 phase 전환마다 직렬화 저장. 프로세스 재시작 시 게임 복원에 사용.

### 4) `ai_histories` — AI 페르소나별 대화 맥락

```sql
CREATE TABLE IF NOT EXISTS ai_histories (
    room_id      TEXT        NOT NULL,
    player_id    TEXT        NOT NULL,
    history_json JSONB       NOT NULL DEFAULT '[]',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, player_id)
);
```

Anthropic 메시지 배열 그대로 저장. 다음 게임 또는 recovery 시 prompt cache 와 함께 재로드.

### 5) `users` — Supabase 사용자 ↔ player_id 매핑

```sql
CREATE TABLE IF NOT EXISTS users (
    auth_id      TEXT PRIMARY KEY,
    player_id    TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

`auth_id` = Supabase JWT 의 `sub` claim. 첫 로그인 시 row 생성, 재로그인은 row 갱신 없음.

### 6) `game_metrics` — Phase A Unit Economics 측정

```sql
CREATE TABLE IF NOT EXISTS game_metrics (
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
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_metrics_started_at ON game_metrics(started_at);
CREATE INDEX IF NOT EXISTS idx_game_metrics_room_id    ON game_metrics(room_id);
```

`AddAIUsage` / `IncrementAdImpression` / `RecordQuickMatch` 가 `INSERT ... ON CONFLICT (game_id) DO UPDATE SET col = col + EXCLUDED.col` 으로 멀티 라이터에 안전하게 누적. Lobby impression 은 일별 sentinel 키 `lobby-YYYY-MM-DD` (room_id = `'lobby'`).

자세한 설계: [`docs/superpowers/specs/2026-04-24-phase-a-unit-economics-foundation-design.md`](docs/superpowers/specs/2026-04-24-phase-a-unit-economics-foundation-design.md) §3-D.

---

## 스키마 변경 절차

새 테이블이 필요하거나 기존 테이블에 컬럼 추가 시:

1. 변경 DDL 을 이 README 에 **추가**한다 (기존 항목 수정이면 인라인 갱신)
2. 변경 의도와 영향을 같은 PR 안에서 [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) "핵심 설계 결정" 에 기록 (스키마 변경이 설계 결정인 경우)
3. 운영 DB 에는 사람이 `psql` 로 직접 적용 — 자동 적용 없음
4. **FK 는 추가하지 않는다** (정책)

---

## 디렉토리 구조

```
ai_side/
├── README.md                ← 이 파일 (DB DDL 포함)
├── docker-compose.yml       ← Postgres + Redis (로컬)
├── backend/
│   ├── cmd/server/          ← 진입점
│   ├── config.example.toml  ← 설정 템플릿 (실제 config.toml 은 gitignored)
│   └── internal/            ← 도메인·핸들러·repo·AI·게임 엔진
├── frontend/
│   ├── .env.example         ← env 템플릿
│   └── src/                 ← React 컴포넌트·페이지·zustand 스토어
└── docs/
    ├── ARCHITECTURE.md      ← 설계 결정·스택·결정 로그
    ├── STATUS.md            ← Phase 체크리스트·서비스 상태·최근 변경
    ├── ROADMAP.md           ← Tier 1/2/3 작업
    └── superpowers/
        ├── specs/           ← 기능별 상세 설계
        └── plans/           ← 기능별 구현 계획
```

---

## 라이선스

TBD.
