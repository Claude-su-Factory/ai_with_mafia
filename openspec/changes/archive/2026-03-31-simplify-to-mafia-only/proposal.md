## Why

이 플랫폼은 원래 여러 게임을 동적으로 지원하는 다중 게임 플랫폼으로 설계되었으나, 실제로는 마피아 게임만 존재하며 새 게임을 추가할 구체적인 계획도 없다. 결과적으로 `GameModule`/`Registry` 추상화는 유지 비용만 치르면서 실질적인 가치를 제공하지 못하고 있으며, 마피아 전용 개념들(`game_type` 필드, `GameConfig` 등)이 공통 레이어에 노출되어 불필요한 복잡성을 만든다.

## What Changes

- **BREAKING** `platform.GameModule` 인터페이스 및 `platform.Registry` 제거
- **BREAKING** `platform.GameConfig` 구조체 제거
- `entity.Room.GameType` 필드 제거
- `dto.CreateRoomRequest.GameType` 필드 제거
- `dto.RoomResponse.GameType` 필드 제거
- `RoomService`에서 `registry` 의존성 제거, 방 생성 시 게임 타입 유효성 검사 제거
- `games/mafia/game.go`의 `MafiaModule` 구조체 및 GameModule 구현 제거, `NewGame` 함수 직접 export
- `cmd/server/main.go`의 `gameManager`에서 registry 제거, mafia 게임 생성 직접 호출
- 프론트엔드 `api.ts`에서 `game_type` 필드 제거

## Capabilities

### New Capabilities
(없음)

### Modified Capabilities
- `platform-core`: GameModule 등록 요구사항 제거, 방 생성에서 게임 타입 검증 제거

## Impact

- `backend/internal/platform/registry.go`: 삭제
- `backend/internal/domain/entity/room.go`: `GameType` 필드 제거
- `backend/internal/domain/dto/room.go`: `GameType` 제거
- `backend/internal/platform/room.go`: `registry` 제거
- `backend/internal/platform/handler.go`: game_type 에러 처리 제거
- `backend/internal/games/mafia/game.go`: `MafiaModule` 제거, `NewGame` export
- `backend/cmd/server/main.go`: registry 제거, 직접 mafia 생성
- `frontend/src/api.ts`: `game_type` 필드 제거
