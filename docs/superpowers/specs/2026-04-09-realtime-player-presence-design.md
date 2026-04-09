# Real-time Player Presence Design

**Date:** 2026-04-09
**Status:** Approved
**Sub-project:** B (실시간 플레이어 현황) — A(UX/Animation), C(광고)는 별도 스펙

---

## Context

현재 대기실에서 다른 플레이어가 입장하거나 퇴장해도 새로고침을 하지 않으면 알 수 없다.
WebSocket 연결 시 `initial_state` 하나만 받고, 이후 플레이어 목록은 갱신되지 않는다.

게임 중 `player_replaced` 이벤트는 이미 broadcast되어 정상 동작하므로 변경하지 않는다.

---

## Goals / Non-Goals

**Goals:**
- 대기실 및 게임 중 플레이어 입장 시 다른 클라이언트에 실시간 반영
- 대기실에서 플레이어 퇴장 시 다른 클라이언트에 실시간 반영
- 입장/퇴장 시 채팅창에 시스템 메시지 표시

**Non-Goals:**
- 로비(방 목록) 페이지 실시간 갱신 (별도 작업)
- 입장/퇴장 토스트 알림 (시스템 메시지로 충분)
- 게임 중 퇴장 처리 변경 (기존 `player_replaced` 로직 유지)

---

## Decisions

### 1. 신규 이벤트 타입 (`backend/internal/domain/entity/game.go`)

```go
const (
    // ... 기존 이벤트들
    EventPlayerJoined GameEventType = "player_joined"
    EventPlayerLeft   GameEventType = "player_left"
)
```

---

### 2. 백엔드 hub.go 변경

#### 2-1. 입장 broadcast

WS 핸들러에서 `initial_state`를 해당 클라이언트에게 전송한 직후, 같은 방의 **다른 클라이언트**에게 broadcast한다.

```go
// hub.go — WS 연결 핸들러, initial_state 전송 직후
h.Broadcast(roomID, dto.GameEventDTO{
    Type: string(entity.EventPlayerJoined),
    Payload: map[string]any{
        "player_id":   playerID,
        "player_name": player.Name,
    },
}, false)
```

`Broadcast`는 호출한 클라이언트 자신에게는 전송하지 않으므로 자기 자신에게 "입장" 메시지가 오는 현상은 발생하지 않는다.

단, 현재 `Broadcast`가 발신자를 제외하지 않는 구현이라면 `playerID`를 제외하는 옵션을 추가하거나, 프론트엔드에서 자신의 `player_id`와 같은 경우 시스템 메시지를 무시한다.

#### 2-2. 퇴장 broadcast

`doRemove`에서 **플레이어 이름을 RemovePlayer 호출 이전에 조회**해두고, 대기실 상태일 때 퇴장 이벤트를 broadcast한다.
(게임 중 퇴장은 기존 `player_replaced` 로직이 처리하므로 중복 broadcast하지 않는다.)

```go
func (h *Hub) doRemove(roomID, playerID string) {
    // 1. 역할 및 이름 사전 조회 (RemovePlayer 전)
    var role entity.Role
    var playerName string
    if room, err := h.roomService.GetByID(roomID); err == nil {
        if p := room.PlayerByID(playerID); p != nil {
            role = p.Role
            playerName = p.Name
        }
    }

    wasPlaying := false
    if room, err := h.roomService.GetByID(roomID); err == nil {
        wasPlaying = room.GetStatus() == entity.RoomStatusPlaying
    }

    room := h.roomService.RemovePlayer(roomID, playerID)

    // ... Redis publish (기존 유지)

    if room == nil {
        // 전원 이탈 처리 (기존 유지)
        h.gameManager.StopGame(roomID)
        h.Broadcast(roomID, dto.GameEventDTO{
            Type:    string(entity.EventGameOver),
            Payload: map[string]any{"reason": "all_humans_left"},
        }, false)
        return
    }

    if wasPlaying {
        // 게임 중 — AI 대체 (기존 유지)
        go func() {
            if err := h.gameManager.ReplaceWithAI(roomID, playerID, role); err != nil {
                h.logger.Warn("ReplaceWithAI failed", ...)
            }
        }()
        h.Broadcast(roomID, dto.GameEventDTO{
            Type:    string(entity.EventPlayerReplaced),
            Payload: map[string]any{"player_id": playerID, "message": "플레이어가 이탈하여 AI로 대체됩니다."},
        }, false)
        return
    }

    // 대기실 퇴장 — player_left broadcast (신규)
    h.Broadcast(roomID, dto.GameEventDTO{
        Type: string(entity.EventPlayerLeft),
        Payload: map[string]any{
            "player_id":   playerID,
            "player_name": playerName,
        },
    }, false)
}
```

---

### 3. 프론트엔드 변경

#### 3-1. `types.ts` — WsEvent 유니온 타입 추가

```ts
export type WsEvent =
  // ... 기존 타입들
  | { type: 'player_joined'; payload: { player_id: string; player_name: string } }
  | { type: 'player_left';   payload: { player_id: string; player_name: string } }
```

#### 3-2. `gameStore.ts` — 이벤트 핸들러 추가

```ts
case 'player_joined': {
  const { player_id, player_name } = event.payload
  // 자기 자신 입장 이벤트는 무시 (initial_state로 이미 처리됨)
  if (player_id === get().playerID) break
  set((s) => ({
    room: s.room
      ? {
          ...s.room,
          players: s.room.players.some((p) => p.id === player_id)
            ? s.room.players
            : [...s.room.players, {
                id: player_id,
                name: player_name,
                is_alive: true,
                is_ai: false,
              }],
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

`WaitingRoom.tsx`는 이미 `room.players`를 store에서 직접 읽으므로 **추가 수정 없이** 실시간 반영된다.

---

## Files Changed

| 파일 | 변경 유형 |
|---|---|
| `backend/internal/domain/entity/game.go` | 수정 (`EventPlayerJoined`, `EventPlayerLeft` 추가) |
| `backend/internal/platform/ws/hub.go` | 수정 (입장 broadcast, doRemove 대기실 분기 추가) |
| `frontend/src/types.ts` | 수정 (WsEvent 유니온 타입 2개 추가) |
| `frontend/src/store/gameStore.ts` | 수정 (`player_joined`, `player_left` 핸들러 추가) |

---

## Risks / Trade-offs

- **자기 자신 입장 이벤트 중복**: `Broadcast`가 발신자 포함 여부에 따라 자신의 입장 메시지가 본인에게 도달할 수 있음. 프론트에서 `player_id === get().playerID` 조건으로 무시하여 방어.
- **재연결 시 중복 입장 메시지**: 짧은 재연결(grace period 내) 시에도 `player_joined`가 broadcast될 수 있음. `room.players`에 이미 존재하는 player_id면 추가하지 않는 중복 방지 로직으로 처리.
- **doRemove 이름 조회 타이밍**: `playerName`을 `RemovePlayer` 호출 전에 조회하므로 race 없음. 현재 `role` 조회 패턴과 동일한 방식.
