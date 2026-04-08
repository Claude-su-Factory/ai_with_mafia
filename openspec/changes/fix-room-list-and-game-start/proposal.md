## Why

방 목록 화면과 게임 시작 흐름에 세 가지 버그가 겹쳐 있어, 방을 만들어도 혼자서 시작할 수 없고 다른 사람도 입장할 수 없는 상태다.

1. 방 목록의 인원수가 항상 0으로 표시된다 — `ListPublic()`이 DB에서 room만 읽고 players를 로드하지 않아서, in-memory 상태를 우회하기 때문이다.
2. 혼자서 게임을 시작할 수 없다 — `WaitingRoom.tsx`에 `players.length >= 2` 조건이 하드코딩되어 있어, AI가 나머지를 채워주는 1인 시작 시나리오를 막는다.
3. 두 번째 사람이 방에 들어올 수 없다 — 방 생성 시 프론트엔드가 `max_humans=1`을 기본값으로 전송하기 때문에, 방장 1명이 입장하면 즉시 정원 초과가 된다.

## What Changes

- `internal/platform/room.go` `ListPublic()`: DB에서 가져온 room 목록을 in-memory 맵과 merge하여 players 데이터가 포함되도록 수정
- `frontend/src/components/WaitingRoom.tsx`: `canStart` 조건을 `players.length >= 2`에서 `players.length >= 1`로 변경
- `frontend/src/api.ts`: `createRoom` 호출 시 `max_humans` 기본값을 `1`에서 `6`으로 변경

## Capabilities

### New Capabilities

없음.

### Modified Capabilities

- `platform-core`: 방 목록 조회 시 현재 참가 인원이 정확히 반영되어야 한다는 요구사항 추가

## Impact

- `internal/platform/room.go`: `ListPublic()` 메서드 수정 (DB 결과 → in-memory merge)
- `frontend/src/components/WaitingRoom.tsx`: `canStart` 조건 1줄 수정
- `frontend/src/api.ts`: `max_humans` 기본값 1줄 수정

## Non-goals

- 방 생성 UI에 max_humans 입력 필드를 추가하지 않는다. 현재 디자인은 항상 6인 게임(AI로 채움)을 기본으로 하므로, max_humans 노출은 불필요하다.
- players를 DB에 별도 테이블로 영속화하지 않는다. 플레이어 목록은 in-memory 상태가 source of truth이며, 게임 중 플레이어 상태는 game_state로 별도 관리된다.
