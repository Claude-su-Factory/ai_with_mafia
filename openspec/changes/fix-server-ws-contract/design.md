## Context

현재 서버와 프론트엔드 사이 WebSocket 이벤트에는 3가지 구조 불일치가 있다.

1. **채팅 이중 경로**: 플레이어 채팅은 `gameEventFunc`를 통해 `{type, payload:{sender_id, sender_name, message}}` 형태로 전달되지만, AI 채팅은 `aiManager` callback에서 직접 `hub.Broadcast`를 호출하여 `{type, player_id, message, mafia_only}` flat 구조로 전달된다. 프론트는 둘 중 하나의 구조만 처리할 수 있다.

2. **night_action 전체 브로드캐스트**: `RecordInvestigation`이 emit하는 `night_action` 이벤트에 `MafiaOnly: false`가 설정되어 있고, `PlayerID`도 없어 모든 WS 클라이언트에 전달된다.

3. **phase_change에 round 누락**: `RunDayDiscussion`, `RunDayVote`, `RunNight` 모두 phase_change payload에 `round` 필드를 포함하지 않는다.

`SendToPlayer`는 이미 `hub.go`에 구현되어 있다.

## Goals / Non-Goals

**Goals:**
- AI 채팅 broadcast를 `payload` 래핑 구조로 통일
- `night_action` 이벤트를 해당 경찰 플레이어에게만 전송
- 모든 `phase_change` 이벤트 payload에 `round` 필드 포함

**Non-Goals:**
- 프론트엔드 코드 변경 (별도 변경으로 처리)
- 채팅 외 다른 이벤트 구조 변경
- 게임 로직 변경

## Decisions

### AI 채팅 구조 통일: gameEventFunc 경유

**결정**: AI 채팅 callback에서 직접 `hub.Broadcast`를 호출하는 대신 `gameEventFunc`를 통해 표준 이벤트 구조로 전달한다.

**대안**: AI callback에서 `{type, payload:{...}}` 구조를 직접 조립해 broadcast.
**왜 기각**: gameEventFunc 우회를 그대로 두면 이후에도 불일치가 재발할 수 있다. gameEventFunc 경유가 더 일관성 있다.

구체적으로 `cmd/server/main.go`의 AI chat callback에서:
```go
// 변경 전
hub.Broadcast(roomID, map[string]any{
    "type": "chat", "player_id": ..., "message": ..., "mafia_only": ...,
})

// 변경 후 — GameEvent를 구성해 gameEventFunc에 전달
evType := entity.EventChat
if mafiaOnly {
    evType = entity.EventMafiaChat
}
gm.gameEventFunc(roomID, entity.GameEvent{
    Type: evType,
    Payload: map[string]any{
        "sender_id":   playerID,
        "sender_name": playerName,
        "message":     message,
    },
    MafiaOnly: mafiaOnly,
})
```

### night_action 단독 전송: SendToPlayer 사용

**결정**: `phases.go`의 `RecordInvestigation`에서 `night_action` emit 시 `PlayerID` 필드를 설정하고, `gameEventFunc`에서 PlayerID가 있는 이벤트는 `hub.SendToPlayer`로 라우팅한다.

**대안**: `gameEventFunc` 바깥에서 직접 `SendToPlayer` 호출.
**왜 기각**: gameEventFunc가 단일 진입점이므로 거기서 처리하는 게 일관성 있다.

`GameEvent` 구조체에 `PlayerID string` 필드를 추가하고:
```go
// phases.go - RecordInvestigation
pm.emit(entity.GameEvent{
    Type:     entity.EventNightAction,
    Payload:  ...,
    PlayerID: policeID,  // 경찰 플레이어 ID
})

// main.go - gameEventFunc
if event.PlayerID != "" {
    hub.SendToPlayer(roomID, event.PlayerID, ...)
} else {
    hub.Broadcast(roomID, ..., event.MafiaOnly)  // MafiaOnly=true이면 마피아에게만, false이면 전체
}
```

### phase_change round: emit 시점에 직접 포함

**결정**: `phases.go`의 각 페이즈 함수가 `phase_change` emit 시 payload에 `"round": state.Round` 필드를 직접 포함한다.

**대안**: `State()` 메서드에서 자동으로 추가.
**왜 기각**: phase_change는 이벤트 emit이지 상태 조회가 아니므로 emit 시점에 명시하는 게 더 명확하다.

## Risks / Trade-offs

- **GameEvent.PlayerID 추가는 기존 emit과 호환**: `PlayerID`가 빈 문자열이면 기존 브로드캐스트 동작을 그대로 유지하므로 다른 이벤트에 영향 없음.
- **AI callback 시그니처 변경 범위**: `AgentOutput`, `SetCallbacks`, main.go 콜백 3곳을 모두 수정해야 한다. `Agent`는 이미 `a.Persona.Name`으로 이름을 알고 있으므로 room.Players 조회는 불필요하다.
