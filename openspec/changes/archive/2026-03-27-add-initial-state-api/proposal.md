## Why

WS 연결/재연결 시 서버가 보내는 초기 상태가 roomID만 담긴 빈 메시지라 클라이언트가 현재 게임 상태를 알 수 없다. 프론트엔드가 페이지 새로고침이나 재연결 후 UI를 복원하려면 phase, 타이머 잔여 시간, 생존자 목록, 내 역할, 투표 현황을 한 번에 받아야 한다.

## What Changes

- `PhaseManager`에 페이즈 시작 시각(`phaseStartedAt`) 기록 추가 → 타이머 잔여 시간 계산 가능
- `entity.GameState`에 `TimerRemainingSeconds` 필드 추가
- `GameManager` 인터페이스에 `GetSnapshot(roomID)` 메서드 추가
- `hub.ServeWS`의 초기 메시지를 `room_state` → `initial_state`로 교체, 방 정보 + 게임 스냅샷 + 이 클라이언트의 역할 포함

## Capabilities

### New Capabilities

- `ws-initial-state`: WS 연결/재연결 직후 클라이언트가 UI를 완전히 복원할 수 있는 초기 상태 이벤트

### Modified Capabilities

없음 — 기존 WS 메시지 타입이 `room_state`에서 `initial_state`로 바뀌지만, 기존 스펙 문서에 정의된 요구사항 수준의 변경은 없음

## Impact

- `internal/games/mafia/phases.go` — `phaseStartedAt` 필드, `State()`에 잔여 시간 포함
- `internal/domain/entity/game.go` — `GameState.TimerRemainingSeconds` 필드 추가
- `internal/platform/ws/hub.go` — `GameManager` 인터페이스 확장, `ServeWS` 초기 메시지 교체
- `cmd/server/main.go` — `gameManager.GetSnapshot()` 구현
