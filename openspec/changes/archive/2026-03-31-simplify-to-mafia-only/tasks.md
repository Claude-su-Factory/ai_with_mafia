## 1. MafiaModule 제거 및 NewGame export

- [x] 1.1 `backend/internal/games/mafia/game.go`에서 `MafiaModule` 구조체 및 `NewModule`, `Name`, `Config`, `NewGame(room)` 메서드 제거
- [x] 1.2 `backend/internal/games/mafia/game.go`에서 `platform` 패키지 import 제거
- [x] 1.3 `backend/internal/games/mafia/game.go`에 `NewGame(room *entity.Room, timers Timers, logger *zap.Logger) *MafiaGame` 함수 추가 (`newGame` 래퍼 export)

## 2. platform/registry.go 제거

- [x] 2.1 `backend/internal/platform/registry.go` 파일 삭제

## 3. entity.Room에서 GameType 제거

- [x] 3.1 `backend/internal/domain/entity/room.go`에서 `GameType string` 필드 제거

## 4. DTO에서 game_type 제거

- [x] 4.1 `backend/internal/domain/dto/room.go`의 `CreateRoomRequest`에서 `GameType string` 필드 제거
- [x] 4.2 `backend/internal/domain/dto/room.go`의 `RoomResponse`에서 `GameType string` 필드 제거

## 5. RoomService에서 registry 의존성 제거

- [x] 5.1 `backend/internal/platform/room.go`의 `RoomService` 구조체에서 `registry *Registry` 필드 제거
- [x] 5.2 `backend/internal/platform/room.go`의 `NewRoomService` 함수에서 `registry *Registry` 파라미터 제거
- [x] 5.3 `backend/internal/platform/room.go`의 `Create` 메서드에서 `registry.Get(req.GameType)` 호출 및 관련 에러 처리 제거
- [x] 5.4 `backend/internal/platform/room.go`의 `Create` 메서드에서 `room.GameType = req.GameType` 대입 제거
- [x] 5.5 `backend/internal/platform/room.go`에서 `totalPlayers = 6` 상수 제거 (사용처 없어짐)
- [x] 5.6 `backend/internal/platform/room.go`의 `ToRoomResponse`에서 `GameType` 필드 대입 제거

## 6. handler.go에서 game_type 관련 코드 제거

- [x] 6.1 `backend/internal/platform/handler.go`의 `createRoom`에서 game type not found 에러 분기 제거

## 7. repository에서 game_type 처리 수정

- [x] 7.1 `backend/internal/repository/room.go`의 `Save`에서 `game_type` 컬럼에 `'mafia'` 리터럴을 전달하도록 수정 (entity.Room.GameType 참조 제거)
- [x] 7.2 `backend/internal/repository/room.go`의 `scanRoom`에서 game_type을 버려지는 로컬 변수 `var gameType string`으로 스캔하도록 수정
- [x] 7.3 `backend/internal/repository/room.go`의 `scanRooms`에서 동일하게 game_type을 버려지는 로컬 변수로 스캔하도록 수정

## 8. main.go에서 gameManager 수정

- [x] 8.1 `backend/cmd/server/main.go`의 `gameManager` 구조체에서 `registry *platform.Registry` 필드 제거
- [x] 8.2 `backend/cmd/server/main.go`의 `newGameManager` 함수에서 `r *platform.Registry` 파라미터 제거
- [x] 8.3 `backend/cmd/server/main.go`의 `gameManager`에 마피아 타이머 설정을 위한 `mafiaCfg *config.MafiaGameConfig` 필드 추가
- [x] 8.4 `backend/cmd/server/main.go`의 `gameManager.start()`에서 `gm.registry.Get(room.GameType)` + `mod.Config()` + `mod.NewGame(room)` 호출을 `mafia.NewGame(room, timers, gm.logger)` 직접 호출로 교체 (timers는 `gm.mafiaCfg`에서 생성)
- [x] 8.5 `backend/cmd/server/main.go`의 `gameManager.start()`에서 `aiCount` 계산을 `mafia.TotalPlayers - room.HumanCount()`로 교체
- [x] 8.6 `backend/cmd/server/main.go`에서 registry 생성·등록 코드 제거, `platform.NewRoomService` 호출에서 registry 인자 제거

## 9. 프론트엔드 수정

- [x] 9.1 `frontend/src/api.ts`의 `createRoom` 요청 body에서 `game_type` 필드 제거
- [x] 9.2 `frontend/src/api.ts`의 `CreateRoomParams` 타입에서 `game_type?: string` 필드 제거

## 10. 컴파일 및 타입 검증

- [x] 10.1 `cd backend && go build ./...`으로 Go 컴파일 오류 없음 확인
- [x] 10.2 `cd frontend && npx tsc --noEmit`으로 TypeScript 오류 없음 확인
