# Bug Fix & Spec Compliance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 인간 플레이어 채팅/투표 버그 수정, AI 밤 행동 수정, 유저 이탈 시 AI 대체 및 게임 종료 로직 구현, 나가기 확인 모달 추가.

**Architecture:** 백엔드는 밤 페이즈 이벤트 구조를 하나로 통합(EventMafiaChannelOpen 분리)하고, GameManager에 StopGame/ReplaceWithAI를 추가한다. 프론트엔드는 WS 액션 페이로드 포맷을 백엔드 DTO에 맞게 수정하고, LeaveConfirmModal을 신규 생성한다.

**Tech Stack:** Go 1.22+, Fiber, Zustand, React Router DOM v7, TypeScript

---

## File Map

| 파일 | 변경 유형 | 이유 |
|------|---------|------|
| `backend/internal/domain/entity/game.go` | 수정 | EventMafiaChannelOpen 타입 추가 |
| `backend/internal/games/mafia/phases.go` | 수정 | RunNight 이벤트 1개로 통합 |
| `backend/internal/games/mafia/phases_test.go` | 수정 | 밤 페이즈 이벤트 구조 테스트 추가 |
| `backend/internal/ai/agent.go` | 수정 | round 타입 수정, EventMafiaChannelOpen 핸들러 |
| `backend/internal/platform/game_manager.go` | 수정 | activeGame에 ctx 추가, StopGame/ReplaceWithAI, drain 루프 제거 |
| `backend/internal/ai/manager.go` | 수정 | AddAgent 추가, 세마포어 ctx 수정 |
| `backend/internal/platform/ws/hub.go` | 수정 | GameManager 인터페이스 확장, doRemove 로직 수정 |
| `frontend/src/components/VotePanel.tsx` | 수정 | 투표 페이로드 포맷 수정 |
| `frontend/src/components/GameRoom.tsx` | 수정 | 마피아 채팅 페이로드 포맷 수정 |
| `frontend/src/components/LeaveConfirmModal.tsx` | 신규 | 나가기 확인 모달 |
| `frontend/src/pages/RoomPage.tsx` | 수정 | LeaveConfirmModal 연결 |

---

## Task 1: entity에 EventMafiaChannelOpen 추가

**Files:**
- Modify: `backend/internal/domain/entity/game.go`

- [ ] **Step 1: EventMafiaChannelOpen 상수 추가**

`backend/internal/domain/entity/game.go`의 const 블록에 추가:

```go
const (
	EventChat             GameEventType = "chat"
	EventPhaseChange      GameEventType = "phase_change"
	EventVote             GameEventType = "vote"
	EventKill             GameEventType = "kill"
	EventNightAction      GameEventType = "night_action"
	EventPlayerLeft       GameEventType = "player_left"
	EventPlayerReplaced   GameEventType = "player_replaced"
	EventGameOver         GameEventType = "game_over"
	EventMafiaChat        GameEventType = "mafia_chat"
	EventTimerUpdate      GameEventType = "timer_update"
	EventMafiaChannelOpen GameEventType = "mafia_channel_open" // 신규
)
```

- [ ] **Step 2: 컴파일 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/domain/entity/game.go
git commit -m "feat: add EventMafiaChannelOpen game event type"
```

---

## Task 2: phases.go — RunNight 이벤트 구조 수정

**Files:**
- Modify: `backend/internal/games/mafia/phases.go`
- Modify: `backend/internal/games/mafia/phases_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`backend/internal/games/mafia/phases_test.go` 파일 끝에 추가:

```go
// --- RunNight event structure ---

func TestRunNight_EmitsSinglePhaseChangeWithAlivePlayers(t *testing.T) {
	players := newTestPlayers()
	pm, eventCh := newTestPM(players)

	// Run with zero timer so it returns immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled immediately so night ends right away

	pm.state.mu.Lock()
	pm.state.Phase = entity.PhaseNight
	pm.state.Round = 2
	pm.state.mu.Unlock()

	pm.RunNight(ctx)

	events := drainEvents(eventCh)

	// Must have exactly one EventPhaseChange and one EventMafiaChannelOpen
	var phaseChanges, mafiaChannelOpens []entity.GameEvent
	for _, e := range events {
		switch e.Type {
		case entity.EventPhaseChange:
			phaseChanges = append(phaseChanges, e)
		case entity.EventMafiaChannelOpen:
			mafiaChannelOpens = append(mafiaChannelOpens, e)
		}
	}

	if len(phaseChanges) != 1 {
		t.Errorf("expected exactly 1 EventPhaseChange, got %d", len(phaseChanges))
	}
	if len(mafiaChannelOpens) != 1 {
		t.Errorf("expected exactly 1 EventMafiaChannelOpen, got %d", len(mafiaChannelOpens))
	}

	// EventPhaseChange must not be MafiaOnly
	if phaseChanges[0].MafiaOnly {
		t.Error("EventPhaseChange should not be MafiaOnly")
	}

	// EventPhaseChange must include alive_players
	ap, ok := phaseChanges[0].Payload["alive_players"]
	if !ok {
		t.Error("EventPhaseChange payload must contain alive_players")
	}
	ids, ok := ap.([]string)
	if !ok || len(ids) == 0 {
		t.Error("alive_players must be a non-empty []string")
	}

	// EventMafiaChannelOpen must be MafiaOnly
	if !mafiaChannelOpens[0].MafiaOnly {
		t.Error("EventMafiaChannelOpen must be MafiaOnly")
	}
}
```

파일 상단에 `"context"` 임포트 추가 (이미 없다면):
```go
import (
	"context"
	"testing"

	"go.uber.org/zap"

	"ai-playground/internal/domain/entity"
)
```

- [ ] **Step 2: 테스트 실패 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./internal/games/mafia/ -run TestRunNight_EmitsSinglePhaseChangeWithAlivePlayers -v
```

Expected: FAIL (현재 두 개의 EventPhaseChange가 emit됨)

- [ ] **Step 3: phases.go RunNight 수정**

`backend/internal/games/mafia/phases.go`의 `RunNight` 함수에서 두 emit 블록을 다음으로 교체:

```go
// RunNight handles night phase: mafia kill + police investigate.
func (pm *PhaseManager) RunNight(ctx context.Context) {
	pm.state.mu.Lock()
	pm.state.Phase = entity.PhaseNight
	pm.state.NightKills = make(map[string]string)
	pm.state.phaseStartedAt = time.Now()
	pm.state.phaseDuration = pm.timers.Night
	pm.state.mu.Unlock()

	pm.save(ctx)

	// 전체 대상 phase_change (alive_players 포함)
	pm.emit(entity.GameEvent{
		Type: entity.EventPhaseChange,
		Payload: map[string]any{
			"phase":         string(entity.PhaseNight),
			"duration":      int(pm.timers.Night.Seconds()),
			"alive_players": pm.aliveIDs(),
			"round":         pm.state.Round,
		},
	})

	// 마피아 전용 채널 오픈 알림
	pm.emit(entity.GameEvent{
		Type:      entity.EventMafiaChannelOpen,
		MafiaOnly: true,
		Payload: map[string]any{
			"message": "밤이 되었습니다. 처치할 대상을 상의하세요.",
		},
	})

	select {
	case <-time.After(pm.timers.Night):
	case <-ctx.Done():
		return
	}

	pm.processMafiaKill()
}
```

- [ ] **Step 4: 테스트 통과 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./internal/games/mafia/ -v
```

Expected: 모든 테스트 PASS

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/games/mafia/phases.go backend/internal/games/mafia/phases_test.go
git commit -m "fix: unify night phase_change event and separate mafia_channel_open"
```

---

## Task 3: agent.go — round 타입 수정 + EventMafiaChannelOpen 핸들러

**Files:**
- Modify: `backend/internal/ai/agent.go`

- [ ] **Step 1: round 타입 단언 수정**

`agent.go`의 `openDiscussion` 함수에서 타입 단언 수정:

```go
func (a *Agent) openDiscussion(ctx context.Context, event entity.GameEvent) {
	round, _ := event.Payload["round"].(int) // float64 → int 수정
	var prompt string
	if round <= 1 {
		prompt = "낮 토론이 시작됐습니다. 게임 첫 라운드입니다. 자연스럽게 첫 인사나 의견을 한 문장으로 말하세요."
	} else {
		prompt = fmt.Sprintf("낮 토론 %d라운드가 시작됐습니다. 지금까지의 대화를 바탕으로 의심되는 점이나 관찰을 한 문장으로 말하세요.", round)
	}

	reply := a.callLLM(ctx, a.cfg.ModelDefault, prompt)
	if reply == "" || strings.TrimSpace(reply) == "[PASS]" {
		return
	}

	a.addHistory(anthropic.NewAssistantMessage(anthropic.NewTextBlock(reply)))
	a.delayedOutput(ctx, AgentOutput{
		PlayerID:   a.PlayerID,
		PlayerName: a.Persona.Name,
		Message:    reply,
		MafiaOnly:  false,
	})
}
```

- [ ] **Step 2: handleEvent에 EventMafiaChannelOpen 케이스 추가**

`agent.go`의 `handleEvent` 함수:

```go
func (a *Agent) handleEvent(ctx context.Context, event entity.GameEvent) {
	switch event.Type {
	case entity.EventChat:
		a.onChat(ctx, event)
	case entity.EventPhaseChange:
		a.onPhaseChange(ctx, event)
	case entity.EventMafiaChat:
		if a.Role == entity.RoleMafia {
			a.onMafiaChat(ctx, event)
		}
	case entity.EventMafiaChannelOpen:
		// 마피아 팀 채널 오픈 알림을 히스토리에 기록
		msg, _ := event.Payload["message"].(string)
		if msg != "" {
			a.addHistory(anthropic.NewUserMessage(anthropic.NewTextBlock(
				"[시스템]: " + msg,
			)))
		}
	}
}
```

- [ ] **Step 3: 컴파일 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/ai/agent.go
git commit -m "fix: correct round type assertion and handle mafia_channel_open event in agent"
```

---

## Task 4: game_manager.go — StopGame, ReplaceWithAI, drain 루프 수정

**Files:**
- Modify: `backend/internal/platform/game_manager.go`

- [ ] **Step 1: activeGame 구조체에 ctx 필드 추가**

`game_manager.go`의 `activeGame` 구조체:

```go
type activeGame struct {
	game   entity.Game
	cancel context.CancelFunc
	ctx    context.Context // 신규: AI 대체 시 사용
}
```

`start()` 함수에서 activeGame 등록 부분 수정:

```go
gm.activeGames[room.ID] = activeGame{game: game, cancel: cancelGame, ctx: gameCtx}
```

- [ ] **Step 2: drain 루프 제거 및 forward 고루틴 수정**

`start()` 함수의 두 고루틴 블록을 다음으로 교체. 기존 첫 번째 고루틴(heartbeat+drain)과 두 번째 고루틴(forward)을 아래와 같이 정리한다:

```go
go func() {
    // Heartbeat to maintain leader lock
    if gm.leaderLock != nil {
        go func() {
            ticker := time.NewTicker(10 * time.Second)
            defer ticker.Stop()
            for {
                select {
                case <-ticker.C:
                    gm.leaderLock.Heartbeat(gameCtx, room.ID)
                case <-gameCtx.Done():
                    return
                }
            }
        }()
    }

    game.Start(gameCtx)
    room.SetStatus(entity.RoomStatusFinished)
    gm.logger.Info("game finished", zap.String("room_id", room.ID))

    cancelGame()

    // Release leader lock and delete checkpoint
    if gm.leaderLock != nil {
        gm.leaderLock.Release(context.Background(), room.ID)
    }
    if gm.gameStateRepo != nil {
        if err := gm.gameStateRepo.Delete(context.Background(), room.ID); err != nil {
            gm.logger.Error("failed to delete game state checkpoint",
                zap.String("room_id", room.ID), zap.Error(err))
        }
    }

    gm.mu.Lock()
    delete(gm.activeGames, room.ID)
    gm.mu.Unlock()
}()

// Forward game events to AI agents and WS clients
go func() {
    for {
        select {
        case event := <-game.Subscribe():
            gm.ai.BroadcastEvent(room.ID, event)

            if event.Type == entity.EventPhaseChange {
                go gm.ai.SaveHistories(context.Background(), room.ID)
            }

            if event.Type == entity.EventGameOver {
                go gm.saveGameResult(room, event, startedAt)
            }

            if gm.GameEventFunc != nil {
                gm.GameEventFunc(room.ID, event)
            }

        case <-gameCtx.Done():
            // Drain remaining events after context cancellation
            for {
                select {
                case event := <-game.Subscribe():
                    gm.ai.BroadcastEvent(room.ID, event)
                    if event.Type == entity.EventGameOver {
                        go gm.saveGameResult(room, event, startedAt)
                    }
                    if gm.GameEventFunc != nil {
                        gm.GameEventFunc(room.ID, event)
                    }
                default:
                    return
                }
            }
        }
    }
}()
```

- [ ] **Step 3: StopGame 메서드 추가**

`game_manager.go` 파일 끝에 추가:

```go
// StopGame cancels the active game context for roomID, stopping the game engine.
// Used when all human players have left (spec #3).
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

- [ ] **Step 4: ReplaceWithAI 메서드 추가**

`game_manager.go` 파일 끝에 추가:

```go
// ReplaceWithAI spawns a new AI agent to take over a human player's slot.
// Called when a human player disconnects during an active game (spec #7).
func (gm *GameManager) ReplaceWithAI(roomID, playerID string, role entity.Role) error {
    gm.mu.Lock()
    ag, ok := gm.activeGames[roomID]
    gm.mu.Unlock()
    if !ok {
        return nil
    }

    personas := gm.personaPool.Assign(1)
    gm.ai.AddAgent(ag.ctx, roomID, playerID, role, personas[0])
    return nil
}
```

- [ ] **Step 5: 컴파일 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 6: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/platform/game_manager.go
git commit -m "feat: add StopGame and ReplaceWithAI to GameManager, remove drain loop race"
```

---

## Task 5: ai/manager.go — AddAgent 추가, 세마포어 수정

**Files:**
- Modify: `backend/internal/ai/manager.go`

- [ ] **Step 1: 세마포어 컨텍스트 수정**

`manager.go`의 `SpawnAgents` 함수 내 출력 처리 고루틴을 수정:

```go
go func(agent *Agent, rID string, ctx context.Context) {
    for {
        select {
        case out, ok := <-agent.Output():
            if !ok {
                return
            }
            select {
            case m.semaphore <- struct{}{}:
                m.handleOutput(rID, out)
                <-m.semaphore
            case <-ctx.Done():
                return
            }
        case <-ctx.Done():
            return
        }
    }
}(a, roomID, ctx)
```

- [ ] **Step 2: AddAgent 메서드 추가**

`manager.go` 파일 끝에 추가:

```go
// AddAgent spawns a single new AI agent for a room at runtime.
// Used when a human player is replaced by AI mid-game (spec #7).
func (m *Manager) AddAgent(ctx context.Context, roomID string, playerID string, role entity.Role, persona Persona) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.agents[roomID] == nil {
        m.agents[roomID] = make(map[string]*Agent)
    }

    // Collect current mafia IDs from existing agents in this room
    mafiaIDs := make([]string, 0)
    for _, a := range m.agents[roomID] {
        if a.Role == entity.RoleMafia {
            mafiaIDs = append(mafiaIDs, a.PlayerID)
        }
    }
    if role == entity.RoleMafia {
        mafiaIDs = append(mafiaIDs, playerID)
    }

    allies := make([]string, 0)
    if role == entity.RoleMafia {
        for _, id := range mafiaIDs {
            if id != playerID {
                allies = append(allies, id)
            }
        }
    }

    a := NewAgent(playerID, persona, role, allies, m.cfg, m.client, m.logger)
    m.agents[roomID][playerID] = a

    go func(agent *Agent, rID string) {
        for {
            select {
            case out, ok := <-agent.Output():
                if !ok {
                    return
                }
                select {
                case m.semaphore <- struct{}{}:
                    m.handleOutput(rID, out)
                    <-m.semaphore
                case <-ctx.Done():
                    return
                }
            case <-ctx.Done():
                return
            }
        }
    }(a, roomID)

    go a.Run(ctx)
}
```

- [ ] **Step 3: 컴파일 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/ai/manager.go
git commit -m "feat: add AddAgent to ai.Manager and fix semaphore goroutine leak"
```

---

## Task 6: hub.go — GameManager 인터페이스 확장 및 doRemove 수정

**Files:**
- Modify: `backend/internal/platform/ws/hub.go`

- [ ] **Step 1: GameManager 인터페이스 확장**

`hub.go`의 `GameManager` 인터페이스:

```go
type GameManager interface {
    StartGame(ctx context.Context, room *entity.Room) error
    RestartGame(ctx context.Context, room *entity.Room) error
    NotifyEvent(roomID string, event entity.GameEvent)
    TryRecover(ctx context.Context, roomID string)
    GetSnapshot(roomID string) *dto.GameSnapshot
    StopGame(roomID string)                                                      // 신규
    ReplaceWithAI(roomID, playerID string, role entity.Role) error               // 신규
}
```

- [ ] **Step 2: doRemove 수정**

`hub.go`의 `doRemove` 함수 전체를 교체:

```go
// doRemove removes the player from the room service and handles game continuation.
func (h *Hub) doRemove(roomID, playerID string) {
    // Get player role before removing (role is lost after RemovePlayer)
    var role entity.Role
    if room, err := h.roomService.GetByID(roomID); err == nil {
        if p := room.PlayerByID(playerID); p != nil {
            role = p.Role
        }
    }

    room := h.roomService.RemovePlayer(roomID, playerID)

    // Notify other instances
    if h.rdb != nil {
        p := pubsubPayload{
            Origin:    h.instanceID,
            EventType: "player_removed",
            PlayerID:  playerID,
        }
        if err := publishToRoom(h.serverCtx, h.rdb, roomID, p); err != nil {
            h.logger.Warn("redis publish player_removed failed",
                zap.String("room_id", roomID),
                zap.String("player_id", playerID),
                zap.Error(err))
        }
    }

    // room == nil means RemovePlayer deleted it (no humans remain)
    if room == nil {
        h.gameManager.StopGame(roomID)
        h.Broadcast(roomID, dto.GameEventDTO{
            Type:    string(entity.EventGameOver),
            Payload: map[string]any{"reason": "all_humans_left"},
        }, false)
        return
    }

    // Game in progress: replace departing player with AI
    if room.GetStatus() == entity.RoomStatusPlaying {
        go func() {
            if err := h.gameManager.ReplaceWithAI(roomID, playerID, role); err != nil {
                h.logger.Warn("ReplaceWithAI failed",
                    zap.String("room_id", roomID),
                    zap.String("player_id", playerID),
                    zap.Error(err))
            }
        }()
    }

    h.Broadcast(roomID, dto.GameEventDTO{
        Type: string(entity.EventPlayerReplaced),
        Payload: map[string]any{
            "player_id": playerID,
            "message":   "플레이어가 이탈하여 AI로 대체됩니다.",
        },
    }, false)
}
```

- [ ] **Step 3: 컴파일 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 4: 전체 백엔드 테스트 실행**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./...
```

Expected: 모든 테스트 PASS

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/platform/ws/hub.go
git commit -m "feat: update hub GameManager interface, implement AI replacement and game stop on player leave"
```

---

## Task 7: 프론트엔드 — 투표 및 마피아 채팅 페이로드 수정

**Files:**
- Modify: `frontend/src/components/VotePanel.tsx`
- Modify: `frontend/src/components/GameRoom.tsx`

- [ ] **Step 1: VotePanel.tsx 투표 페이로드 수정**

`frontend/src/components/VotePanel.tsx`의 `handleVote` 함수:

```tsx
function handleVote(targetID: string) {
  sendAction('vote', { vote: { target_id: targetID } })
}
```

(기존: `sendAction('vote', { target_id: targetID })`)

- [ ] **Step 2: GameRoom.tsx 마피아 채팅 페이로드 수정**

`frontend/src/components/GameRoom.tsx`의 `handleMafiaChat` 함수:

```tsx
function handleMafiaChat() {
  const trimmed = mafiaText.trim()
  if (!trimmed) return
  sendAction('chat', { chat: { message: trimmed, mafia_only: true } })
  setMafiaText('')
}
```

(기존: `sendAction('chat', { message: trimmed, mafia_only: true })`)

- [ ] **Step 3: 타입스크립트 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/components/VotePanel.tsx frontend/src/components/GameRoom.tsx
git commit -m "fix: correct vote and mafia chat WS action payload format"
```

---

## Task 8: 프론트엔드 — LeaveConfirmModal 생성 및 RoomPage 연결

**Files:**
- Create: `frontend/src/components/LeaveConfirmModal.tsx`
- Modify: `frontend/src/pages/RoomPage.tsx`

- [ ] **Step 1: LeaveConfirmModal.tsx 생성**

`frontend/src/components/LeaveConfirmModal.tsx` 신규 파일:

```tsx
import { useBlocker } from 'react-router-dom'
import { useGameStore } from '../store/gameStore'

const T = {
  bg: 'rgba(0,0,0,0.75)',
  surface: '#181410',
  surfaceBorder: '#2E2820',
  text: '#ECE7DE',
  textMuted: '#786F62',
  danger: '#8C1F1F',
  dangerDim: 'rgba(140,31,31,0.15)',
}
const SANS = "'DM Sans', system-ui, sans-serif"
const MONO = "'JetBrains Mono', monospace"

export default function LeaveConfirmModal() {
  const { room, result, disconnect } = useGameStore()

  const shouldBlock = room?.status === 'playing' && !result

  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      shouldBlock && currentLocation.pathname !== nextLocation.pathname
  )

  if (blocker.state !== 'blocked') return null

  function handleLeave() {
    disconnect()
    blocker.proceed()
  }

  function handleStay() {
    blocker.reset()
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 9000,
        background: T.bg,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
      onClick={handleStay}
    >
      <div
        style={{
          background: T.surface,
          border: `1px solid ${T.surfaceBorder}`,
          borderRadius: '4px',
          padding: '32px',
          maxWidth: '400px',
          width: '90%',
          fontFamily: SANS,
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{
          fontFamily: MONO, fontSize: '10px', color: T.textMuted,
          textTransform: 'uppercase', letterSpacing: '0.12em',
          marginBottom: '16px',
        }}>
          게임 이탈
        </div>

        <p style={{ color: T.text, fontSize: '15px', lineHeight: 1.6, margin: '0 0 8px' }}>
          게임에서 나가시겠습니까?
        </p>
        <p style={{ color: T.textMuted, fontSize: '13px', lineHeight: 1.6, margin: '0 0 28px' }}>
          나가면 AI가 당신의 역할을 대신하며 게임은 계속 진행됩니다.
        </p>

        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
          <button
            onClick={handleStay}
            style={{
              background: 'transparent', color: T.textMuted,
              border: `1px solid ${T.surfaceBorder}`, borderRadius: '2px',
              padding: '9px 20px', fontSize: '13px', fontFamily: SANS,
              cursor: 'pointer',
            }}
          >
            계속 플레이
          </button>
          <button
            onClick={handleLeave}
            style={{
              background: T.dangerDim, color: T.danger,
              border: `1px solid ${T.danger}40`, borderRadius: '2px',
              padding: '9px 20px', fontSize: '13px', fontFamily: SANS,
              cursor: 'pointer',
            }}
          >
            나가기
          </button>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: RoomPage.tsx에 LeaveConfirmModal 추가**

`frontend/src/pages/RoomPage.tsx` 전체를 다음으로 교체:

```tsx
import { useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useGameStore } from '../store/gameStore'
import WaitingRoom from '../components/WaitingRoom'
import GameRoom from '../components/GameRoom'
import ResultOverlay from '../components/ResultOverlay'
import LeaveConfirmModal from '../components/LeaveConfirmModal'

export default function RoomPage() {
  const { id: roomID } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { connect, disconnect, room, result } = useGameStore()

  useEffect(() => {
    if (!roomID) return
    const playerID = localStorage.getItem(`player_id_${roomID}`)
    if (!playerID) {
      navigate('/lobby')
      return
    }
    connect(roomID)
    return () => {
      disconnect()
    }
  }, [roomID])

  if (!room) {
    return (
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        minHeight: '100vh', background: '#0E0C09', color: '#786F62',
        fontFamily: "'JetBrains Mono', monospace", fontSize: '11px',
        textTransform: 'uppercase', letterSpacing: '0.1em',
      }}>
        CONNECTING...
      </div>
    )
  }

  return (
    <div style={{ minHeight: '100vh', background: '#0E0C09', position: 'relative' }}>
      {room.status === 'waiting' || room.status === 'finished' ? (
        <WaitingRoom />
      ) : (
        <GameRoom />
      )}
      {result && <ResultOverlay />}
      <LeaveConfirmModal />
    </div>
  )
}
```

- [ ] **Step 3: 타입스크립트 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/components/LeaveConfirmModal.tsx frontend/src/pages/RoomPage.tsx
git commit -m "feat: add leave confirmation modal for in-game navigation"
```

---

## 최종 검증

- [ ] **백엔드 전체 빌드 및 테스트**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./... && go test ./...
```

Expected: 에러 없음, 모든 테스트 PASS

- [ ] **프론트엔드 전체 빌드**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **통합 확인 체크리스트**

다음 동작을 수동으로 확인한다:

1. 게임 대기실에서 "시작" → 게임 시작됨
2. 낮 토론 페이즈에서 채팅 입력 → 채팅이 다른 플레이어에게 표시됨
3. 투표 페이즈에서 플레이어 클릭 → 투표 집계가 실시간으로 반영됨
4. 밤 페이즈에서 경찰로 조사 클릭 → 조사 결과가 채팅창에 표시됨
5. 밤 페이즈에서 마피아로 처치 클릭 → 대상 플레이어 사망 처리됨
6. 게임 진행 중 뒤로가기 → 확인 모달 표시됨
7. 모달에서 "나가기" → 로비로 이동, 게임은 AI가 이어서 진행
8. 모든 인간 플레이어 이탈 → game_over 이벤트 수신

---

## 스펙 커버리지 체크

| 스펙 항목 | 구현 태스크 |
|---------|-----------|
| F1: 인간 투표 무시 | Task 7 (VotePanel 페이로드 수정) |
| F2: 인간 채팅 무시 | Task 7 (GameRoom 마피아채팅 수정) |
| B1: 경찰 AI 조사 안 함 | Task 2 (RunNight 이벤트 통합) |
| B2: 마피아 AI decideKill 2회 | Task 2 (RunNight 이벤트 통합) |
| B5: 라운드 타입 단언 | Task 3 (agent.go round 타입) |
| 스펙 #3: 유저 0명 게임 종료 | Task 6 (hub doRemove StopGame) |
| 스펙 #6: 나가기 모달 | Task 8 (LeaveConfirmModal) |
| 스펙 #7: AI 대체 | Task 4+5+6 (ReplaceWithAI) |
| B3: drain 루프 race | Task 4 (game_manager drain 제거) |
| B4: 세마포어 고루틴 리크 | Task 5 (manager semaphore 수정) |
