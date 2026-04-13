# User Profile & Stats Design

**Date:** 2026-04-13
**Status:** Approved

---

## Context

Google OAuth 로그인 도입 이후 사용자 신원이 고정되었으나, 닉네임은 여전히 방 입장 시마다 입력하는 방식이다. 이를 Google 계정 1:1 고정 닉네임으로 전환하고, 승패 통계와 게임 기록을 볼 수 있는 프로필 페이지를 추가한다.

JWT 관리는 Supabase 클라이언트가 현행대로 담당한다 (변경 없음).

---

## Section 1: 닉네임 관리

### 현재 문제

- 방 만들기/참가 시마다 닉네임을 직접 입력
- `users` 테이블의 `display_name`은 Google 실명이 초기값으로 들어가지만 이후 활용되지 않음

### 변경

- 방 입장 시 닉네임 입력 필드 제거
- 백엔드의 `createRoom` / `joinRoom` / `joinByCode`가 `users` 테이블의 `display_name`을 자동으로 읽어 사용
- 닉네임 수정은 `/profile` 페이지에서만 가능
- 최초 로그인 시 기본값: Google 계정 실명 (`user_metadata.full_name`)

---

## Section 2: 백엔드 API

### `GET /api/me` (기존 확장)

```json
{
  "player_id": "uuid",
  "display_name": "김철수"
}
```

### `PUT /api/me` (신규)

닉네임 수정.

```json
// Request
{ "display_name": "새닉네임" }

// Response 200
{ "player_id": "uuid", "display_name": "새닉네임" }

// Response 400 — 빈 문자열이거나 50자 초과
{ "error": "invalid display_name" }
```

### `GET /api/me/stats` (신규)

`game_result_players` 테이블에서 집계. AI 플레이어(`is_ai = true`) 제외.

```json
{
  "total_games": 42,
  "wins": 25,
  "losses": 17,
  "win_rate": 0.595,
  "by_role": {
    "mafia":   { "games": 14, "wins": 9,  "win_rate": 0.643 },
    "citizen": { "games": 20, "wins": 12, "win_rate": 0.600 },
    "police":  { "games": 8,  "wins": 4,  "win_rate": 0.500 }
  }
}
```

게임 없을 때: 모든 수치 0, `win_rate` 0.

### `GET /api/me/games?limit=20` (신규)

최근 게임 기록. 기본 limit 20, 최대 50.

```json
[
  {
    "game_id": "uuid",
    "played_at": "2026-04-13T10:00:00Z",
    "role": "mafia",
    "survived": true,
    "won": true,
    "round_count": 5,
    "duration_sec": 420
  }
]
```

`won` 계산: `game_results.winner_team`과 플레이어 역할을 비교.
- `winner_team == "mafia"` && `role == "mafia"` → `won = true`
- `winner_team == "citizen"` && `role != "mafia"` → `won = true`
- 그 외 → `won = false`

---

## Section 3: 프론트엔드

### 라우트

`/profile` 추가 (`frontend/src/main.tsx`).

### authStore 변경

`displayName: string` 필드 추가. `initialize()` 및 `onAuthStateChange`에서 `/api/me` 응답의 `display_name`을 저장.

### LobbyPage 변경

- 헤더에 닉네임 + 프로필 아이콘 → 클릭 시 `/profile` 이동
- 방 만들기 모달: 닉네임 입력 필드 제거
- 방 참가 모달: 닉네임 입력 필드 제거
- 코드 참가 모달: 닉네임 입력 필드 제거
- API 호출 시 `player_name` 파라미터 제거

### ProfilePage (`/profile`)

**헤더 영역**
- Google 프로필 사진 (`user.user_metadata.avatar_url`, 없으면 기본 아바타)
- 닉네임 + 수정 버튼 (연필 아이콘) → 클릭 시 인라인 텍스트 필드로 전환, 저장/취소 버튼
- Google 이메일 (표시 전용)

**통계 카드 (4개)**
```
[총 게임]  [승]  [패]  [승률]
```

**역할별 통계 테이블**

| 역할 | 게임 | 승 | 패 | 승률 |
|------|------|----|----|------|
| 마피아 | 14 | 9 | 5 | 64.3% |
| 시민 | 20 | 12 | 8 | 60.0% |
| 경찰 | 8 | 4 | 4 | 50.0% |

플레이 기록 없는 역할은 행 표시 안 함.

**최근 게임 기록 테이블** (최대 20개, 세로 스크롤)

| 날짜 | 역할 | 결과 | 생존 | 라운드 | 게임시간 |
|------|------|------|------|--------|---------|
| 04-13 | 마피아 | 승 | O | 5R | 7분 |

`duration_sec`을 분 단위로 표시 (예: 420 → "7분").

**하단 버튼**
- "로비로 돌아가기" → `navigate('/lobby')`
- "로그아웃" → `signOut()` → `navigate('/')`

---

## Files Changed

| 파일 | 변경 유형 | 내용 |
|------|----------|------|
| `backend/internal/repository/user.go` | 수정 | `UpdateDisplayName`, `GetDisplayName` 추가 |
| `backend/internal/repository/game_result.go` | 수정 | `GetStatsByPlayerID`, `GetRecentGamesByPlayerID` 추가 |
| `backend/internal/platform/handler.go` | 수정 | `PUT /api/me`, `GET /api/me/stats`, `GET /api/me/games` 추가; `GET /api/me` display_name 포함; createRoom/joinRoom/joinByCode에서 display_name 자동 사용 |
| `frontend/src/store/authStore.ts` | 수정 | `displayName` 필드 추가 |
| `frontend/src/main.tsx` | 수정 | `/profile` 라우트 추가 |
| `frontend/src/pages/LobbyPage.tsx` | 수정 | 닉네임 입력 제거, 헤더에 프로필 진입 버튼 추가 |
| `frontend/src/api.ts` | 수정 | `updateMe`, `getMyStats`, `getMyGames` 함수 추가; `createRoom`/`joinRoom`/`joinByCode` 파라미터에서 `player_name` 제거 |
| `frontend/src/pages/ProfilePage.tsx` | 신규 | 프로필 페이지 전체 |

---

## Risks / Trade-offs

- **닉네임 제거 후 방 내 표시**: 게임 중 플레이어 이름은 방 입장 시 백엔드가 `users` 테이블에서 읽어온 `display_name`을 그대로 사용. 게임 중 닉네임을 바꿔도 현재 게임에는 반영 안 됨 (다음 방부터 적용).
- **stats 집계 성능**: `game_result_players` 테이블에 player_id 인덱스가 없으면 풀스캔 발생. 마이그레이션에서 인덱스 추가 불필요 (데이터 규모상 허용 가능). 향후 필요시 인덱스 추가.
- **player_name 파라미터 제거**: 기존 `createRoom`/`joinRoom` API 스펙 변경. 프론트엔드와 함께 동시 배포 필요.
