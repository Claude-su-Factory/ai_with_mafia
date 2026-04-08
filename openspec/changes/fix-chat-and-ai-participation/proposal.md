## Why

게임의 핵심 기능인 채팅과 AI 참여가 두 가지 버그로 인해 완전히 동작하지 않으며, 빈 방이 목록에 계속 남는 데이터 정합성 문제도 겹쳐 있다.

1. **채팅 payload 구조 불일치**: 프론트엔드가 `{ type: "chat", message: "..." }` 형태로 전송하지만 백엔드는 `{ type: "chat", chat: { message: "..." } }` 구조를 기대한다. `NotifyEvent()`에서 `req.Chat == nil`이면 즉시 return하기 때문에, 모든 채팅 메시지가 게임 엔진과 AI에 전달되지 않는다.

2. **AI 자발 발언 없음**: `onPhaseChange()`에 `PhaseDayDiscussion` 핸들러가 없어서, 낮 토론 시작 시 AI 에이전트가 아무런 행동도 하지 않는다. 누군가 먼저 말을 걸어야 반응하는데, 버그 1 때문에 채팅 자체가 안 들어오니 게임 내내 침묵한다.

3. **빈 방 목록 잔류**: `RoomService.RemovePlayer()`가 방이 비면 메모리에서만 삭제하고 DB에서는 삭제하지 않는다. `ListPublic()`은 DB를 기준으로 조회하므로 플레이어가 없는 방이 계속 목록에 표시된다.

4. **Rate limit 과잉 억제**: `hub.go`의 채팅 rate limit이 500ms로 설정되어 있어 연속 메시지가 드롭된다. 게임 채팅 특성상 짧은 연속 발언이 자연스러운 상황이므로 완화가 필요하다.

## What Changes

- `frontend/src/components/ChatInput.tsx`: `sendAction` 호출 시 메시지를 `chat` 키 아래 중첩 구조로 전송
- `backend/internal/ai/agent.go`: `onPhaseChange()`에 `PhaseDayDiscussion` 케이스 추가 — 낮 토론 시작 시 AI가 첫 발언 생성
- `backend/internal/platform/room.go`: `RemovePlayer()`에서 방이 비어 메모리에서 삭제할 때 `roomRepo.Delete()`도 호출
- `backend/internal/platform/ws/hub.go`: rate limit을 500ms에서 200ms로 완화

## Capabilities

### New Capabilities

없음.

### Modified Capabilities

- `ai-agent`: 낮 토론 페이즈 시작 시 AI 에이전트가 자발적으로 첫 발언을 생성해야 한다는 요구사항 추가
- `platform-core`: 방에 플레이어가 없으면 DB에서도 즉시 삭제되어야 한다는 요구사항 추가

## Impact

- `frontend/src/components/ChatInput.tsx`: `sendAction` 호출 1줄 수정
- `backend/internal/ai/agent.go`: `onPhaseChange()` switch에 케이스 추가 (~20줄)
- `backend/internal/platform/room.go`: `RemovePlayer()` 내 `roomRepo.Delete()` 호출 추가
- `backend/internal/platform/ws/hub.go`: `msgRateLimit` 상수 수정 1줄

## Non-goals

- AI의 채팅 전략(얼마나 자주 발언할지, 어떤 내용을 말할지)을 정교하게 튜닝하지 않는다. 이번 변경은 "낮 시작 시 첫 발언을 하는지 여부"만 수정한다.
- 게임 진행 중 방 삭제 처리는 변경하지 않는다. 게임이 끝난 후 `finished` 상태로 남는 방은 `ListPublic()` 쿼리의 `status != 'finished'` 조건이 이미 필터링하고 있다.
