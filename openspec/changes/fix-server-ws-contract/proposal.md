## Why

프론트엔드와 서버 간 WebSocket 이벤트 구조 불일치 3가지가 실제 게임 플레이를 불가능하게 만든다. 채팅이 렌더링되지 않고, 경찰 조사 결과가 모든 플레이어에게 노출되며, 라운드 카운터가 항상 1에 고정된다.

## What Changes

- **채팅 이벤트 구조 통일**: AI 채팅 broadcast를 `{type, payload:{sender_id, sender_name, message}}` 형태로 통일 (현재 AI는 flat 구조 사용)
- **`night_action` 경찰에게만 전송**: RecordInvestigation의 `night_action` 이벤트를 전체 broadcast에서 경찰 단독 전송으로 변경 (**BREAKING** for any client expecting broadcast)
- **`phase_change`에 `round` 필드 추가**: RunDayDiscussion, RunDayVote, RunNight의 phase_change payload에 현재 round 값 포함

## Capabilities

### New Capabilities

없음.

### Modified Capabilities

- `ws-event-chat`: AI 채팅과 플레이어 채팅의 payload 구조를 `{sender_id, sender_name, message}` 로 통일
- `ws-event-phase-change`: phase_change payload에 `round` 필드 추가
- `ws-event-night-action`: night_action 이벤트를 전체 broadcast에서 해당 경찰 플레이어에게만 단독 전송으로 변경

## Impact

- `internal/games/mafia/phases.go`: phase_change payload에 round 추가, night_action emit 방식 변경
- `internal/ai/manager.go` 또는 `cmd/server/main.go`: AI 채팅 callback의 broadcast 구조 변경
- `internal/platform/ws/hub.go`: SendToPlayer 기반 targeted delivery 지원 (이미 구현됨)
- 프론트엔드 `gameStore.ts`: 통일된 chat payload 처리 로직으로 업데이트 필요 (별도 frontend 변경)
