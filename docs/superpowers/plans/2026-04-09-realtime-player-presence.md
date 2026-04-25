# Real-time Player Presence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 대기실에서 플레이어가 입장/퇴장할 때 다른 플레이어들이 새로고침 없이 실시간으로 확인할 수 있도록 WebSocket 이벤트를 추가한다.

**Architecture:** 백엔드 `hub.go`에서 WS 연결 수립 직후 `player_joined`를, 대기실 퇴장 시 `player_left`를 broadcast한다. 프론트엔드 gameStore가 두 이벤트를 수신해 `room.players`를 갱신하고 시스템 메시지를 채팅에 추가한다. `WaitingRoom.tsx`는 이미 store에서 직접 읽으므로 추가 수정 없이 자동 반영된다.

**Tech Stack:** Go, Fiber WebSocket, React, TypeScript, Zustand

---

## File Map

| 파일 | 변경 | 책임 |
|---|---|---|
| `backend/internal/domain/entity/game.go` | 수정 | `EventPlayerJoined` 상수 추가 (`EventPlayerLeft`는 이미 존재) |
| `backend/internal/platform/ws/hub.go` | 수정 | 입장 broadcast + doRemove 대기실 퇴장 분기 |
| `frontend/src/types.ts` | 수정 | WsEvent 유니온 타입 2개 추가 |
| `frontend/src/store/gameStore.ts` | 수정 | `player_joined`, `player_left` 핸들러 |

---

## Task 1: EventPlayerJoined 이벤트 타입 추가

**Files:**
- Modify: `backend/internal/domain/entity/game.go`

- [ ] **Step 1: 현재 상수 블록 확인**

```bash
grep -n "EventPlayer" /Users/yuhojin/Desktop/ai_side/backend/internal/domain/entity/game.go
```

Expected: `EventPlayerLeft`는 있고 `EventPlayerJoined`는 없음.

- [ ] **Step 2: EventPlayerJoined 추가**

`backend/internal/domain/entity/game.go`의 const 블록에서 `EventPlayerLeft` 바로 위에 추가:

```go
EventPlayerJoined   GameEventType = "player_joined"
EventPlayerLeft     GameEventType = "player_left"
```

- [ ] **Step 3: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/domain/entity/game.go
git commit -m "feat: add EventPlayerJoined game event type"
```

---

## Task 2: hub.go — 입장/퇴장 broadcast

**Files:**
- Modify: `backend/internal/platform/ws/hub.go`

> 현재 `doRemove`는 `playerRole`과 `wasPlaying`을 이미 RemovePlayer 전에 조회하고 있다. `playerName`만 추가로 조회하면 된다.
> `Broadcast`는 발신자 포함 전체 클라이언트에게 전송하므로, 자기 자신의 `player_joined` 이벤트도 받는다. 프론트엔드에서 필터링한다(Task 3).

- [ ] **Step 1: WS 핸들러에서 initial_state 전송 직후 player_joined broadcast 추가**

`hub.go`의 WS 핸들러에서 `initialStateMsg`를 클라이언트에 전송한 뒤 바로 아래에 추가:

```go
// initial_state를 본인에게 전송 (기존 코드)
if err := client.conn.WriteMessage(websocket.TextMessage, initialStateMsg); err != nil {
    // ... 기존 에러 처리
}

// 같은 방 다른 클라이언트에게 입장 알림
h.Broadcast(roomID, dto.GameEventDTO{
    Type: string(entity.EventPlayerJoined),
    Payload: map[string]any{
        "player_id":   playerID,
        "player_name": player.Name,
    },
}, false)
```

- [ ] **Step 2: doRemove에서 playerName 사전 조회 추가**

현재 `doRemove` 상단의 `playerRole`, `wasPlaying` 조회 블록에 `playerName` 추가:

```go
var playerRole entity.Role
var playerName string
var wasPlaying bool
if preRoom, err := h.roomService.GetByID(roomID); err == nil && preRoom != nil {
    if p := preRoom.PlayerByID(playerID); p != nil {
        playerRole = p.Role
        playerName = p.Name
    }
    wasPlaying = preRoom.GetStatus() == entity.RoomStatusPlaying
}
```

- [ ] **Step 3: doRemove 끝에 대기실 퇴장 broadcast 추가**

현재 `doRemove` 마지막의 `if wasPlaying` 블록 다음에 추가:

```go
if wasPlaying && playerRole != "" {
    // ... 기존 AI 대체 코드 (유지)
    return
}

// 대기실 퇴장 — player_left broadcast
h.Broadcast(roomID, dto.GameEventDTO{
    Type: string(entity.EventPlayerLeft),
    Payload: map[string]any{
        "player_id":   playerID,
        "player_name": playerName,
    },
}, false)
```

- [ ] **Step 4: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 5: 기존 테스트 통과 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./...
```

Expected: 모든 테스트 PASS

- [ ] **Step 6: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/platform/ws/hub.go
git commit -m "feat: broadcast player_joined on connect and player_left on waiting room exit"
```

---

## Task 3: 프론트엔드 — types.ts + gameStore.ts 핸들러

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/store/gameStore.ts`

- [ ] **Step 1: types.ts에 WsEvent 타입 추가**

`frontend/src/types.ts`의 `WsEvent` 유니온 타입 마지막에 추가:

```ts
export type WsEvent =
  | { type: 'initial_state'; payload: { room: Room; game: GameSnapshot | null; my_role: Role } }
  | { type: 'role_assigned'; payload: { role: Role } }
  | { type: 'phase_change'; payload: { phase: Phase; round?: number; duration?: number; alive_players?: string[] } }
  | { type: 'chat'; payload: { sender_id: string; sender_name: string; message: string; mafia_only?: boolean } }
  | { type: 'mafia_chat'; payload: { sender_id: string; sender_name: string; message: string } }
  | { type: 'vote'; payload: { voter_id?: string; target_id?: string; result?: string; votes?: Record<string, string> } }
  | { type: 'kill'; payload: { player_id: string; role?: string; reason?: string } }
  | { type: 'night_action'; payload: { type: string; target_id: string; is_mafia: boolean } }
  | { type: 'game_over'; payload: GameOverResult }
  | { type: 'player_replaced'; payload: { player_id: string; message: string } }
  | { type: 'player_joined'; payload: { player_id: string; player_name: string } }
  | { type: 'player_left'; payload: { player_id: string; player_name: string } }
```

- [ ] **Step 2: gameStore.ts에 player_joined 핸들러 추가**

`gameStore.ts`의 `ws.onmessage` switch 블록에서 `case 'player_replaced':` 다음에 추가:

```ts
case 'player_joined': {
  const { player_id, player_name } = event.payload
  // 자기 자신의 입장 이벤트는 무시 (initial_state로 이미 처리됨)
  if (player_id === get().playerID) break
  set((s) => ({
    room: s.room
      ? {
          ...s.room,
          players: s.room.players.some((p) => p.id === player_id)
            ? s.room.players
            : [
                ...s.room.players,
                { id: player_id, name: player_name, is_alive: true, is_ai: false },
              ],
        }
      : null,
    messages: [
      ...s.messages,
      {
        id: `${Date.now()}-${Math.random()}`,
        player_id: 'system',
        message: `${player_name}님이 입장했습니다.`,
        mafia_only: false,
        is_system: true,
      },
    ],
  }))
  break
}
```

- [ ] **Step 3: gameStore.ts에 player_left 핸들러 추가**

`case 'player_joined':` 바로 다음에 추가:

```ts
case 'player_left': {
  const { player_id, player_name } = event.payload
  set((s) => ({
    room: s.room
      ? {
          ...s.room,
          players: s.room.players.filter((p) => p.id !== player_id),
        }
      : null,
    messages: [
      ...s.messages,
      {
        id: `${Date.now()}-${Math.random()}`,
        player_id: 'system',
        message: `${player_name}님이 퇴장했습니다.`,
        mafia_only: false,
        is_system: true,
      },
    ],
  }))
  break
}
```

- [ ] **Step 4: 프론트엔드 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/types.ts frontend/src/store/gameStore.ts
git commit -m "feat: handle player_joined and player_left events in gameStore"
```

---

## 최종 검증

- [ ] 백엔드 서버 실행 후 두 브라우저 탭에서 같은 방 접속
- [ ] 세 번째 탭으로 입장 → 다른 두 탭의 대기실 플레이어 목록에 즉시 추가되고 "xxx님이 입장했습니다." 메시지 확인
- [ ] 탭 닫기 → 다른 탭에서 플레이어 목록에서 제거되고 "xxx님이 퇴장했습니다." 메시지 확인
