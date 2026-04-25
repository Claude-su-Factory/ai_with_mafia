# Bug Fix: Data Router Migration + Orphan Game State Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `useBlocker` 오류를 유발하는 legacy BrowserRouter를 createBrowserRouter로 마이그레이션하고, 서버 재시작 시 반복되는 고아 game_state WARN 로그를 자동 삭제로 해결한다.

**Architecture:** 프론트엔드는 `main.tsx`에 라우트를 직접 정의하고 `App.tsx`를 삭제한다. 백엔드는 `recoverOrphanGames`에서 room not found 시 해당 `game_states` 레코드를 DB에서 삭제한다. 두 수정은 완전히 독립적이다.

**Tech Stack:** React Router v7, Go, PostgreSQL

---

## File Map

| 파일 | 변경 | 책임 |
|---|---|---|
| `frontend/src/main.tsx` | 수정 | `createBrowserRouter` + `RouterProvider`로 교체, 라우트 직접 정의 |
| `frontend/src/App.tsx` | 삭제 | 라우트가 main.tsx로 이동하므로 불필요 |
| `backend/cmd/server/main.go` | 수정 | `recoverOrphanGames`에 orphan game_state 삭제 로직 추가 |

---

## Task 1: 프론트엔드 — createBrowserRouter 마이그레이션

**Files:**
- Modify: `frontend/src/main.tsx`
- Delete: `frontend/src/App.tsx`

- [ ] **Step 1: App.tsx import 사용처 확인**

```bash
grep -rn "from.*App\|import App" /Users/yuhojin/Desktop/ai_side/frontend/src/
```

Expected: `main.tsx:5:import App from './App.tsx'` 하나만 나와야 함. 다른 파일에서 App을 import하면 그 파일도 수정 필요.

- [ ] **Step 2: main.tsx 전체 교체**

`frontend/src/main.tsx`를 다음으로 교체:

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

- [ ] **Step 3: App.tsx 삭제**

```bash
rm /Users/yuhojin/Desktop/ai_side/frontend/src/App.tsx
```

- [ ] **Step 4: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음. `App.tsx` 관련 import 에러가 없어야 함.

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/main.tsx
git rm frontend/src/App.tsx
git commit -m "fix: migrate to createBrowserRouter to support useBlocker"
```

---

## Task 2: 백엔드 — 고아 game_state 자동 삭제

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 현재 recoverOrphanGames 확인**

```bash
grep -n "room not found\|continue" /Users/yuhojin/Desktop/ai_side/backend/cmd/server/main.go
```

Expected: `logger.Warn("recoverOrphanGames: room not found", ...)` + `continue` 패턴 확인.

- [ ] **Step 2: room not found 분기에 삭제 로직 추가**

`backend/cmd/server/main.go`의 `recoverOrphanGames` 함수에서 다음 블록을:

```go
		room, err := roomSvc.GetByID(state.RoomID)
		if err != nil {
			logger.Warn("recoverOrphanGames: room not found", zap.String("room_id", state.RoomID))
			continue
		}
```

다음으로 교체:

```go
		room, err := roomSvc.GetByID(state.RoomID)
		if err != nil {
			// Room no longer exists — delete the orphan game_state to prevent repeated warnings on restart.
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

- [ ] **Step 3: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 4: 기존 테스트 통과 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./...
```

Expected: 모든 테스트 PASS

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/cmd/server/main.go
git commit -m "fix: delete orphan game_state records in recoverOrphanGames"
```

---

## 최종 검증

- [ ] 프론트엔드: 브라우저에서 방 생성 후 `/rooms/:id` 진입 시 `useBlocker` 오류 없음
- [ ] 프론트엔드: 게임 진행 중 뒤로가기 시 LeaveConfirmModal 정상 표시
- [ ] 백엔드: 서버 재시작 후 `recoverOrphanGames: room not found` WARN 로그 없음 (기존 고아 레코드가 삭제됐으므로)
- [ ] 백엔드: 정상 game_state는 그대로 복구됨 (room이 존재하는 경우)
