# Supabase Auth + Presence Fix Design

**Date:** 2026-04-12
**Status:** Approved

---

## Context

두 가지 독립적인 문제를 하나의 스펙에서 다룬다.

1. **인증 부재** — 현재 player_id는 방 입장 시 백엔드가 생성하는 랜덤 UUID. 사용자 신원이 없으므로 같은 사람이 여러 탭/기기에서 동시에 같은 방에 들어갈 수 있고, 새로고침 시 player_id가 localStorage에 없으면 방을 잃어버린다.

2. **뒤로가기 즉각 반영 안 됨** — 대기실에서 플레이어가 뒤로가기를 누르면 WebSocket이 닫히지만, grace timer(재접속 허용 대기) 때문에 수 초 뒤에야 `player_left` 이벤트가 브로드캐스트된다.

---

## Section 1: 인증 아키텍처

### 선택: Supabase Google OAuth + JWT 검증

Supabase는 **인증 계층**만 담당한다. 기존 Go WebSocket 인프라, Redis, PostgreSQL은 유지.

### 전체 흐름

```
[LandingPage] Google 로그인 버튼
    → Supabase OAuth redirect → Google 동의 → 콜백
    → Supabase JWT 발급 (access_token, sub = auth_id)
    → Supabase 클라이언트가 localStorage에 세션 저장
    → 백엔드 호출 시 Authorization: Bearer {jwt}
    → 백엔드 미들웨어: Supabase JWKS로 JWT 검증 → sub 추출 = auth_id
    → users 테이블에서 auth_id로 player_id 조회 (없으면 신규 생성)
```

### Google 계정 = 고정 player_id

- `users` 테이블: `auth_id (Supabase UUID)` → `player_id (UUID, 영구 고정)`, `display_name`
- 최초 로그인 시 생성, 이후 항상 동일한 player_id 반환
- 게임 내부 로직(투표, AI 교체, role 배정)은 player_id 기준 유지 — 변경 없음
- localStorage `player_id_{roomID}` 패턴 제거, player_id는 인증에서 가져옴

### WebSocket 토큰 전달

브라우저 WebSocket은 커스텀 헤더 미지원. 쿼리 파라미터로 전달:

```
/ws/rooms/{roomID}?token={supabase_jwt}
```

현재 `?player_id=` 파라미터를 `?token=`으로 교체. 허브에서 JWT 검증 후 player_id 추출.

---

## Section 2: 활성 세션 추적 + 중복 접속 방지

### Redis 세션 키

```
Key:   user_session:{player_id}
Value: {room_id}
TTL:   24시간
```

### 방 입장 흐름 (create / join / join-by-code 공통)

```
1. JWT 검증 → player_id 추출
2. Redis: GET user_session:{player_id}
3. 값 있음 →
   a. DB에서 해당 room 존재 확인
   b. 존재: 409 + { existing_room_id } 반환
   c. 없음 (stale): Redis 키 삭제 후 입장 허용
4. 값 없음 → 입장 허용
5. 입장 성공 → Redis SET user_session:{player_id} = room_id
```

### 프론트엔드 409 처리

```
"이미 입장한 방이 있습니다.
 돌아가시겠습니까?"    [돌아가기]  [취소]
```

`[돌아가기]` → `navigate('/rooms/{existing_room_id}')`
`[취소]` → 현재 화면 유지

### 세션 해제 시점

- 명시적 나가기 버튼 (LeaveConfirmModal 확인)
- 게임 종료 후 나가기 버튼 (ResultOverlay)
- `pagehide` sendBeacon 수신 (Section 3)
- 방 삭제

---

## Section 3: 뒤로가기 즉각 반영 (`pagehide` + `sendBeacon`)

### 문제

`useBlocker`는 게임 중(`status === 'playing'`)에만 동작. 대기실에서는 뒤로가기를 막지 않음. 뒤로가기 → WS 닫힘 → grace timer → 수 초 후 `player_left`. 방에 즉각 반영 안 됨.

### 수정

**프론트엔드 `RoomPage.tsx`에 `pagehide` 핸들러 추가:**

```typescript
useEffect(() => {
  const handlePageHide = () => {
    navigator.sendBeacon(`/api/rooms/${roomID}/leave`)
  }
  window.addEventListener('pagehide', handlePageHide)
  return () => window.removeEventListener('pagehide', handlePageHide)
}, [roomID])
```

`navigator.sendBeacon`: 페이지 언로드 중에도 HTTP POST를 신뢰성 있게 전송. 뒤로가기, 탭 닫기, 새 URL 이동 모두 커버. Authorization 헤더 필요 없음 — 요청 body에 player_id 포함.

**백엔드 `POST /api/rooms/:id/leave` (새 엔드포인트):**

- JWT 검증 없이 body의 player_id 사용 (sendBeacon은 헤더 커스터마이징이 제한적)
- Hub에서 해당 플레이어 즉시 제거 (`doRemove()` 직접 호출, grace timer 없음)
- Redis `user_session:{player_id}` 삭제
- `player_left` 이벤트 즉시 브로드캐스트

**grace timer는 유지:** 새로고침처럼 의도치 않은 WS 끊김은 재접속을 허용. sendBeacon이 도착하면 grace timer를 취소하고 즉시 제거. Hub에 `ForceRemove(playerID string)` 메서드 추가.

---

## Files Changed

| 파일 | 변경 유형 | 내용 |
|------|----------|------|
| `backend/migrations/000006_create_users.up.sql` | 신규 | `users` 테이블: auth_id, player_id, display_name, created_at |
| `backend/migrations/000006_create_users.down.sql` | 신규 | DROP TABLE users |
| `backend/internal/repository/user.go` | 신규 | UserRepository: GetByAuthID, Upsert |
| `backend/internal/platform/auth.go` | 신규 | JWT 검증 미들웨어 (Supabase JWKS) |
| `backend/internal/platform/handler.go` | 수정 | leave 엔드포인트 추가, 입장 시 세션 체크, player_id를 JWT에서 추출 |
| `backend/internal/repository/redis.go` | 수정 | SetUserSession, GetUserSession, DeleteUserSession 추가 |
| `backend/internal/platform/ws/hub.go` | 수정 | `?token=` 파라미터 처리, ForceRemove 메서드 추가 |
| `backend/cmd/server/main.go` | 수정 | UserRepo 주입, auth 미들웨어 등록 |
| `frontend/src/lib/supabase.ts` | 신규 | Supabase 클라이언트 초기화 |
| `frontend/src/store/authStore.ts` | 신규 | Supabase 세션 상태 (user, signIn, signOut) |
| `frontend/src/pages/LandingPage.tsx` | 수정 | Google 로그인 버튼; 이미 Supabase 세션이 있으면 `/lobby`로 자동 리디렉트 |
| `frontend/src/store/gameStore.ts` | 수정 | player_id를 authStore에서 가져옴, WS URL에 token 파라미터 |
| `frontend/src/api.ts` | 수정 | Authorization: Bearer 헤더 추가 |
| `frontend/src/pages/RoomPage.tsx` | 수정 | pagehide sendBeacon 핸들러 |
| `frontend/.env.development` | 수정 | VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY |
| `frontend/.env.production` | 수정 | VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY |

---

## Risks / Trade-offs

- **sendBeacon 인증**: `navigator.sendBeacon`은 커스텀 헤더를 지원하지 않아 JWT 검증 없이 body의 player_id를 신뢰해야 한다. 악의적 요청으로 타인을 강제 퇴장시킬 수 있는 위험이 있다. 이 엔드포인트는 내부 UUID를 알아야 하므로 실질적 위험은 낮지만, 향후 HMAC 서명 등으로 강화 가능.
- **grace timer 유지**: 새로고침 시 재접속 경험 보호를 위해 기존 grace 로직은 유지. sendBeacon과 grace timer가 경쟁 조건(race)이 될 수 있으나, ForceRemove가 먼저 처리되면 grace timer는 이미 제거된 플레이어를 무시하도록 처리.
- **Supabase JWKS 캐싱**: 매 요청마다 JWKS를 fetch하면 성능 이슈. jwks-go 라이브러리 사용으로 캐싱 처리.
