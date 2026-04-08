## Context

현재 백엔드는 `platform.GameModule` 인터페이스 + `platform.Registry`를 통해 여러 게임을 런타임에 등록·조회하는 구조다. `gameManager.start()`는 `registry.Get(room.GameType)`으로 모듈을 꺼내서 `mod.Config()`로 플레이어 수를 가져오고, `mod.NewGame(room)`으로 게임 인스턴스를 생성한다.

문제는 이 추상화가 실제로 다중 게임을 처리하지 않는다는 점이다. `mafia` 하나만 등록되어 있고, `entity/game.go`에 마피아 전용 Phase 상수와 EventType 상수가 "공통" 레이어에 섞여 있다. `GameType` 필드는 Room entity와 DTO 전반에 걸쳐 불필요하게 전파된다.

## Goals / Non-Goals

**Goals:**
- `GameModule`, `GameConfig`, `Registry` 추상화 제거
- `GameType` 필드를 Room entity, DTO, 방 생성 API에서 제거
- `gameManager`가 mafia 게임을 직접 생성하도록 변경
- `MafiaModule` 구조체 제거, 게임 생성 팩토리 단순화

**Non-Goals:**
- Phase/EventType 상수를 mafia 패키지로 이동 — `entity.GameState.Phase` 타입이 entity 레이어에 있어야 순환 의존 없이 사용 가능하므로 현 위치 유지
- 게임 로직 자체 변경
- DB 스키마 변경 (rooms 테이블의 game_type 컬럼은 그대로 유지, 단순히 backend에서 쓰지 않을 뿐)

## Decisions

### 결정 1: `MafiaModule` 대신 `mafia.NewGame` 직접 호출

현재: `main.go` → `registry.Get("mafia")` → `mod.NewGame(room)`
변경: `main.go` → `mafia.NewGame(room, timers, logger)` 직접 호출

`MafiaModule.NewGame()`이 내부적으로 config에서 Timers를 읽어 `newGame(room, timers, logger)`를 호출하는 구조였다. 이를 풀어서 `gameManager`가 `cfg.Game.Mafia` 설정에서 직접 Timers를 구성하고 `mafia.NewGame()`을 호출한다.

**대안 고려:** MafiaModule은 유지하되 Registry만 제거. 하지만 그러면 불필요한 인터페이스(Name, Config)가 남아 목적이 불명확해진다. 완전히 제거하는 것이 더 명확하다.

### 결정 2: AI 수 계산을 상수로 고정

현재: `gameCfg.TotalPlayers - room.HumanCount()`에서 `gameCfg.TotalPlayers`는 `GameConfig`에서 옴
변경: `mafia.TotalPlayers` 상수를 직접 참조 (`roles.go`에 이미 `const TotalPlayers = 6` 정의)

### 결정 3: `entity.Room.GameType` 제거

Room entity에서 `GameType`을 제거한다. DB의 `game_type` 컬럼은 건드리지 않는다(zero-risk). 마피아만 존재하므로 방 조회 시 game_type은 의미 없는 값이다.

**주의:** `repository/room.go`에서 DB에서 Room을 로드할 때 `game_type` 컬럼을 스캔하는 코드가 있을 수 있다. 이 경우 해당 컬럼 스캔을 제거하거나 무시해야 한다.

### 결정 4: 프론트엔드 `game_type` 필드 제거

`api.ts`의 `createRoom`에서 `game_type: 'mafia'`를 body에서 제거한다. 백엔드가 무시하더라도 불필요한 필드는 제거한다.

## Risks / Trade-offs

- [DB schema 충돌] `repository/room.go`가 DB에서 game_type을 읽어서 `Room.GameType`에 할당하는 코드가 있을 수 있다. entity에서 필드 제거 시 컴파일 오류 발생. → `repository/room.go`도 함께 수정 필요.
- [BREAKING 변경] `game_type`이 방 생성 API 응답에서 사라진다. 외부 클라이언트가 있다면 영향. → 현재 프론트엔드는 `types.ts`의 `Room` 타입에 `game_type`이 없으므로 무영향.

## Migration Plan

1. 백엔드 변경 → 컴파일 확인
2. 프론트엔드 변경 → TypeScript 타입 오류 없음 확인
3. 로컬에서 방 생성/참가/게임 시작 흐름 수동 테스트
