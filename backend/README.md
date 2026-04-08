# AI Playground Backend

## 실행 방법

### 1. 환경 변수 설정

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

### 2. PostgreSQL 시작

프로젝트 루트(`ai_side/`)에서:

```bash
docker-compose up -d
```

```
[+] Running 1/1
 ✔ Container ai_playground_db  Started
```

### 3. 서버 실행

`backend/` 디렉토리에서:

```bash
go run ./cmd/server/
```

```
{"level":"info","msg":"migrations applied"}
{"level":"info","msg":"database connected"}
{"level":"info","msg":"server starting","addr":":3000"}
```

### 4. 헬스 체크

```bash
curl http://localhost:3000/health
```

```json
{"status":"ok"}
```

---

## API 요약

| Method | Path | 설명 |
|--------|------|------|
| POST | /api/rooms | 방 생성 |
| GET  | /api/rooms | 공개 방 목록 |
| GET  | /api/rooms/:id | 방 상세 |
| POST | /api/rooms/:id/join | 방 참가 |
| POST | /api/rooms/join/code | 코드로 참가 |
| POST | /api/rooms/:id/start | 게임 시작 (방장) |
| POST | /api/rooms/:id/restart | 재시작 (방장) |
| GET  | /ws/rooms/:id?player_id=xxx | WebSocket 연결 |

### 방 생성 예시

```bash
curl -X POST http://localhost:3000/api/rooms \
  -H "Content-Type: application/json" \
  -H "X-Player-Name: 플레이어1" \
  -d '{"name":"테스트방","game_type":"mafia","max_humans":2,"visibility":"public"}'
```

### 게임 시작

```bash
curl -X POST http://localhost:3000/api/rooms/{room_id}/start \
  -H "X-Player-ID: {player_id}"
```

---

## WebSocket 이벤트

### 클라이언트 → 서버

```json
{"type":"chat","chat":{"message":"안녕하세요"}}
{"type":"vote","vote":{"target_id":"player-xxx"}}
{"type":"kill","night":{"action_type":"kill","target_id":"player-xxx"}}
```

### 서버 → 클라이언트

```json
{"type":"phase_change","payload":{"phase":"day_discussion","duration":300}}
{"type":"chat","payload":{"player_id":"...","message":"..."}}
{"type":"vote","payload":{"voter_id":"...","target_id":"..."}}
{"type":"kill","payload":{"player_id":"...","role":"mafia","reason":"vote"}}
{"type":"game_over","payload":{"winner":"citizen"}}
{"type":"player_replaced","payload":{"player_id":"...","message":"AI로 대체됩니다."}}
```
