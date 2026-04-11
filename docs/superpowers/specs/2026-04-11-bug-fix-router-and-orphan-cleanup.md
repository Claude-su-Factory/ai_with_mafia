# Bug Fix: Data Router Migration + Orphan Game State Cleanup

**Date:** 2026-04-11
**Status:** Approved

---

## Context

두 가지 독립적인 버그를 함께 수정한다.

1. **프론트엔드 라우터 오류** — `useBlocker must be used within a data router`. `main.tsx`가 `BrowserRouter` (legacy component router)를 사용 중인데 `useBlocker`는 `createBrowserRouter` (data router)에서만 동작한다.

2. **백엔드 고아 game_state 경고** — 서버 시작 시 `recoverOrphanGames: room not found` WARN 로그. `game_states` 테이블에 rooms 테이블에서 이미 삭제된 room_id 레코드가 남아 있어서 발생한다.

---

## Bug 1: Data Router 마이그레이션

### 원인

`frontend/src/main.tsx`가 `BrowserRouter`를 사용하고 있다. `useBlocker` (React Router v6.4+)는 `createBrowserRouter`로 생성된 data router 내에서만 동작한다.

### Fix

`main.tsx`에서 라우트 정의를 직접 담당하도록 변경하고, `App.tsx`를 삭제한다.

**`frontend/src/main.tsx`** — 변경:

```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import './index.css'
import LandingPage from './pages/LandingPage'
import LobbyPage from './pages/LobbyPage'
import RoomPage from './pages/RoomPage'

const router = createBrowserRouter([
  { path: '/', element: <LandingPage /> },
  { path: '/lobby', element: <LobbyPage /> },
  { path: '/rooms/:id', element: <RoomPage /> },
])

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
)
```

**`frontend/src/App.tsx`** — 삭제. 라우트 정의가 `main.tsx`로 이동하므로 불필요.

`RoomPage.tsx`의 `useBlocker` 코드는 변경 없음.

---

## Bug 2: 고아 game_state 자동 삭제

### 원인

`recoverOrphanGames`가 `game_states` 테이블의 모든 레코드를 가져와 복구를 시도한다. 그런데 `rooms` 테이블에서 이미 삭제된 room_id가 `game_states`에 남아 있으면 `roomSvc.GetByID` 실패 후 WARN 로그만 찍고 계속 진행한다. 이 stale 레코드는 서버를 재시작할 때마다 반복적으로 경고를 발생시킨다.

### Fix

**`backend/cmd/server/main.go`** — `recoverOrphanGames` 함수에서 room not found 시 해당 game_state 레코드를 삭제한다:

```go
room, err := roomSvc.GetByID(state.RoomID)
if err != nil {
    // 고아 레코드 삭제 — room이 삭제됐지만 game_state가 남은 경우
    if delErr := gameStateRepo.Delete(ctx, state.RoomID); delErr != nil {
        logger.Warn("recoverOrphanGames: failed to delete orphan state",
            zap.String("room_id", state.RoomID), zap.Error(delErr))
    } else {
        logger.Info("recoverOrphanGames: deleted orphan game state",
            zap.String("room_id", state.RoomID))
    }
    continue
}
```

삭제 성공 시 INFO, 삭제 실패 시 WARN. 다음 재시작부터 해당 room_id에 대한 경고가 발생하지 않는다.

---

## Files Changed

| 파일 | 변경 유형 |
|---|---|
| `frontend/src/main.tsx` | 수정 — createBrowserRouter + RouterProvider로 교체 |
| `frontend/src/App.tsx` | 삭제 |
| `backend/cmd/server/main.go` | 수정 — recoverOrphanGames에 orphan 삭제 로직 추가 |

---

## Risks / Trade-offs

- **App.tsx 삭제**: `App.tsx`를 import하는 다른 파일이 없는지 확인 필요. 현재 `main.tsx`에서만 import되므로 안전.
- **고아 삭제 영구성**: 삭제 후 복구 불가. 단, game_state는 진행 중인 게임의 체크포인트이므로 room이 없으면 복구 의미 없음.
- **동시성**: `recoverOrphanGames`는 서버 시작 시 단일 goroutine에서 실행되므로 race 없음.
