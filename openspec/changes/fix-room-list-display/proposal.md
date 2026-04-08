## Why

방 목록이 두 가지 이유로 잘못된 정보를 표시한다.

1. **스테일 DB 레코드 노출**: `ListPublic()`이 DB에서 방 목록을 가져온 뒤 in-memory 상태와 merge할 때, 메모리에 없는 방(플레이어가 모두 나가서 이미 삭제됐거나 서버 재시작 전 잔존 레코드)도 그대로 반환한다. 결과적으로 실제 사용자가 한 명도 없는 방이 목록에 계속 남는다.

2. **인원수에 AI 포함**: `ToRoomResponse()`가 AI 플레이어를 포함한 전체 players를 반환한다. 게임이 진행 중인 방이 목록에 표시될 때 "6명"처럼 AI를 포함한 숫자가 노출된다. AI는 인원 부족 시 자동으로 채우는 역할이므로, 실제 사용자 수만 표시해야 한다.

## What Changes

- `internal/platform/room.go` `ListPublic()`: DB 결과 merge 시 메모리에 존재하는 방만 포함 (메모리에 없는 방은 제외)
- `internal/platform/room.go` `ToRoomResponse()`: players 목록에서 AI 플레이어 제외

## Capabilities

### New Capabilities

없음.

### Modified Capabilities

- `platform-core`: 방 목록은 현재 메모리에 활성 상태인 방만 포함해야 한다는 요구사항 추가
- `platform-core`: 방 목록의 참가자 수는 실제 사용자만 포함해야 한다는 요구사항 추가

## Impact

- `internal/platform/room.go`: `ListPublic()` merge 루프 수정 (~3줄), `ToRoomResponse()` players 필터 추가 (~2줄)

## Non-goals

- 스테일 DB 레코드를 일괄 정리하는 마이그레이션은 작성하지 않는다. `ListPublic()`에서 메모리 기준으로 필터링하면 자연스럽게 노출이 차단된다.
- 방 상세 조회(`GetByID`)나 WS 연결 시 players 응답은 변경하지 않는다. AI 포함 전체 정보가 필요한 컨텍스트이기 때문이다.
