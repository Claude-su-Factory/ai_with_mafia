# Bug Fix & Spec Compliance Design

**Date:** 2026-04-08  
**Status:** Approved

---

## Context

백엔드 초기 구현과 프론트엔드 구현이 완료됐으나 다음 문제들이 존재한다:

1. 인간 플레이어의 채팅과 투표가 백엔드에서 전부 무시된다 (페이로드 포맷 불일치)
2. 경찰 AI가 조사를 실행하지 않는다 (밤 이벤트에 alive_players 누락)
3. 마피아 AI가 밤마다 decideKill을 2번 호출한다 (이벤트 중복 수신)
4. 유저 이탈 시 AI 대체가 실제로 동작하지 않는다 (메시지만 전송)
5. 유저 전원 이탈 시 게임 엔진이 계속 실행된다 (컨텍스트 미취소)
6. 방 나가기 전 확인 모달이 없다

---

## Goals / Non-Goals

**Goals:**
- 인간 플레이어의 채팅, 투표, 밤 행동이 정상 동작하도록 수정
- 경찰/마피아 AI가 올바르게 동작하도록 수정
- 유저 이탈 시 실제 AI가 해당 슬롯을 이어받아 게임 계속 진행
- 실제 유저 0명 시 게임 강제 종료 및 방 삭제
- 게임 진행 중 나가기 전 확인 모달 표시
- 저수준 고루틴 안정성 개선

**Non-Goals:**
- 랜딩 페이지 집계 UI (추후 별도 작업)
- 새로운 게임 기능 추가
- DB 스키마 변경

---

## Decisions

### 1. 프론트엔드 액션 페이로드 수정 (F1, F2)

백엔드 `ActionRequest` DTO 구조에 맞게 페이로드를 래핑한다.

```ts
// 채팅
sendAction('chat', { chat: { message, mafia_only } })

// 투표
sendAction('vote', { vote: { target_id } })

// kill / investigate — 이미 올바른 형식, 변경 없음
sendAction('kill', { night: { action_type: 'kill', target_id } })
sendAction('investigate', { night: { action_type: 'investigate', target_id } })
```

변경 파일: `VotePanel.tsx`, `GameRoom.tsx`, `ChatInput.tsx`

---

### 2. 밤 페이즈 이벤트 구조 개편 (B1, B2)

기존에 두 개의 `EventPhaseChange` 이벤트를 emit하던 것을 하나로 통합한다.
마피아 전용 채널 알림은 별도 `EventMafiaChannelOpen` 타입으로 분리한다.

```go
// entity 타입 추가
EventMafiaChannelOpen GameEventType = "mafia_channel_open"
```

`phases.go`의 `RunNight` 수정:
```go
// 하나의 EventPhaseChange (전체 대상, alive_players 포함)
emit(EventPhaseChange, MafiaOnly=false, {
    "phase": "night",
    "duration": ...,
    "alive_players": pm.aliveIDs(),
    "round": pm.state.Round,
})

// 마피아 전용 채널 오픈 알림 (별도 이벤트)
emit(EventMafiaChannelOpen, MafiaOnly=true, {
    "message": "밤이 되었습니다. 처치할 대상을 상의하세요.",
})
```

`ai/manager.go`의 `BroadcastEvent`는 기존과 동일하게 `MafiaOnly` 필터링 유지.

`agent.go`에서 `EventMafiaChannelOpen` 처리:
```go
case entity.EventMafiaChannelOpen:
    // 마피아 팀 알림 — chat history에 추가만, 별도 액션 없음
```

결과:
- 경찰 agent는 `EventPhaseChange`(with alive_players) 수신 → `decideInvestigate` 정상 동작
- 마피아 agent는 `EventPhaseChange` 1회만 수신 → `decideKill` 1번만 호출
- 프론트엔드는 기존 `phase_change` 이벤트 처리 로직 그대로 유지 (alive_players 있을 때만 업데이트하는 조건 이미 있음)

---

### 3. AI 라운드 타입 수정 (B5)

`agent.go`의 round 타입 단언 수정:
```go
// 기존
round, _ := event.Payload["round"].(float64)

// 수정
round, _ := event.Payload["round"].(int)
```

---

### 4. 유저 이탈 시 AI 대체 (스펙 #7)

`ws.Hub`의 `GameManager` 인터페이스에 `ReplaceWithAI` 추가:

```go
type GameManager interface {
    // 기존 메서드들...
    ReplaceWithAI(ctx context.Context, roomID, playerID string) error
    StopGame(roomID string)
}
```

`GameManager.ReplaceWithAI` 구현 (`game_manager.go`):
1. `activeGames[roomID]` 확인 — 없으면 return
2. `room.PlayerByID(playerID)`로 이탈 플레이어의 Role 확인
3. `personaPool.Assign(1)`로 새 페르소나 1개 할당
4. 이탈한 플레이어를 `IsAI=true`로 변경 (같은 playerID 유지, Role 유지)
   - room의 player 슬롯 재사용 → 게임 엔진의 `pm.state.Players`에 영향 없음
5. `ai.Manager.AddAgent(gameCtx, roomID, player, persona)`로 새 에이전트 등록 및 실행

`ai/manager.go`에 `AddAgent` 신규 메서드 추가:
```go
func (m *Manager) AddAgent(ctx context.Context, roomID string, player *entity.Player, persona Persona) {
    // 기존 SpawnAgents 로직 일부 추출
    // mafiaIDs를 현재 agents map에서 수집
    // 새 Agent 생성 후 고루틴 시작
}
```

`hub.go`의 `doRemove` 수정:
```go
func (h *Hub) doRemove(roomID, playerID string) {
    room := h.roomService.RemovePlayer(roomID, playerID)

    if h.rdb != nil { /* Redis publish — 기존 유지 */ }

    if room == nil {
        return
    }

    if room.HumanCount() == 0 {
        // 스펙 #3: 유저 전원 이탈 → 게임 종료
        h.gameManager.StopGame(roomID)
        h.roomService.DeleteRoom(roomID)
        h.Broadcast(roomID, dto.GameEventDTO{
            Type:    string(entity.EventGameOver),
            Payload: map[string]any{"reason": "all_humans_left"},
        }, false)
        return
    }

    // 스펙 #7: 게임 중이면 AI 대체
    if room.GetStatus() == entity.RoomStatusPlaying {
        go h.gameManager.ReplaceWithAI(h.serverCtx, roomID, playerID)
    }

    h.Broadcast(roomID, dto.GameEventDTO{
        Type:    string(entity.EventPlayerReplaced),
        Payload: map[string]any{
            "player_id": playerID,
            "message":   "플레이어가 이탈하여 AI로 대체됩니다.",
        },
    }, false)
}
```

---

### 5. 유저 전원 이탈 시 게임 강제 종료 (스펙 #3)

`GameManager.StopGame` 구현:
```go
func (gm *GameManager) StopGame(roomID string) {
    gm.mu.Lock()
    ag, ok := gm.activeGames[roomID]
    if ok {
        ag.cancel()
        delete(gm.activeGames, roomID)
    }
    gm.mu.Unlock()
}
```

`RoomService.DeleteRoom` 신규 추가:
```go
func (s *RoomService) DeleteRoom(roomID string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.rooms, roomID)
}
```

게임 결과는 저장하지 않는다 (스펙 #4 — 비정상 종료 제외).

---

### 6. 나가기 확인 모달 (스펙 #6)

신규 컴포넌트 `LeaveConfirmModal.tsx`:
- 게임 진행 중(`room.status === 'playing'`)이고 `result`가 없을 때만 활성화
- React Router v6의 `useBlocker`로 navigation 차단
- 메시지: "게임에서 나가시겠습니까? 나가면 AI가 당신의 역할을 대신하며 게임은 계속 진행됩니다."
- 버튼: "계속 플레이" (취소) / "나가기" (확인 → disconnect → 이동)
- 대기실(`waiting`) 및 게임 종료 후(`result` 있음)에는 모달 없이 자유 이동

`RoomPage.tsx`에 모달 연결.

---

### 7. 고루틴 안정성 개선 (B3, B4)

**B3** — `game_manager.go` drain 루프 제거:

`game.Start()` 반환 후 drain 루프가 forward 고루틴과 같은 채널을 경쟁하는 문제를 해결한다.
drain 루프를 제거하고 `cancelGame()`을 `game.Start()` 반환 직후 호출한다.
forward 고루틴은 `gameCtx.Done()`이 선택된 후에도 채널에 남은 이벤트를 처리하도록 루프 종료 시 non-blocking drain을 추가한다.

```go
go func() {
    // heartbeat...
    game.Start(gameCtx)
    room.SetStatus(entity.RoomStatusFinished)
    cancelGame()  // ← drain 루프 없이 즉시 취소
    // cleanup...
}()

go func() {
    for {
        select {
        case event := <-game.Subscribe():
            // 이벤트 처리...
        case <-gameCtx.Done():
            // 남은 이벤트 drain
            for {
                select {
                case event := <-game.Subscribe():
                    // 처리
                default:
                    return
                }
            }
        }
    }
}()
```

**B4** — `manager.go` 세마포어 컨텍스트 처리:

```go
select {
case m.semaphore <- struct{}{}:
    m.handleOutput(rID, out)
    <-m.semaphore
case <-ctx.Done():
    return
}
```

---

## Risks / Trade-offs

- **같은 playerID로 AI 교체**: `pm.state.Players`가 playerID로 플레이어를 찾으므로, 같은 ID를 재사용하면 게임 엔진 수정 없이 교체 가능. 단, 해당 플레이어가 이미 사망한 경우 AI가 생성되지만 게임에서 행동할 일이 없으므로 무해.
- **StopGame 후 RemovePlayer 경쟁**: `StopGame`과 `RemovePlayer`가 각각 `activeGames` map과 `rooms` map을 별도로 잠그므로 race 없음.
- **useBlocker 안정성**: React Router v6의 `useBlocker`는 stable API. 브라우저 뒤로가기는 `beforeunload` 이벤트로 별도 처리 필요.

---

## Files Changed

| 파일 | 변경 유형 |
|------|---------|
| `frontend/src/components/VotePanel.tsx` | 수정 |
| `frontend/src/components/GameRoom.tsx` | 수정 |
| `frontend/src/components/ChatInput.tsx` | 수정 |
| `frontend/src/pages/RoomPage.tsx` | 수정 |
| `frontend/src/components/LeaveConfirmModal.tsx` | 신규 |
| `backend/internal/domain/entity/game.go` | 수정 (EventMafiaChannelOpen 추가) |
| `backend/internal/games/mafia/phases.go` | 수정 (RunNight 이벤트 구조) |
| `backend/internal/ai/agent.go` | 수정 (round 타입, EventMafiaChannelOpen 핸들러) |
| `backend/internal/ai/manager.go` | 수정 (AddAgent, 세마포어 수정) |
| `backend/internal/platform/game_manager.go` | 수정 (ReplaceWithAI, StopGame, drain 루프 제거) |
| `backend/internal/platform/room.go` | 수정 (DeleteRoom 추가) |
| `backend/internal/platform/ws/hub.go` | 수정 (doRemove 로직, GameManager 인터페이스) |
