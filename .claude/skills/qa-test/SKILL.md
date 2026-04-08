---
name: qa-test
description: "AI 마피아 게임 QA 및 테스트 스킬. 백엔드-프론트엔드 경계면 검증, Go 단위/통합 테스트 작성·실행, payload 불일치 탐지를 수행한다. 테스트 코드 작성, 버그 검증, 통합 테스트, QA 요청 시 반드시 이 스킬을 사용할 것. 다시 테스트, 테스트 추가, 검증 요청 시에도 이 스킬을 사용."
---

# QA Test Skill

AI 마피아 게임 플랫폼의 통합 검증 및 테스트 작업을 위한 가이드.

## 작업 흐름

### 경계면 검증 요청 시

1. 백엔드 DTO 파일 읽기: `backend/internal/domain/dto/room.go`
2. 프론트 타입 파일 읽기: `frontend/src/types.ts`
3. WS 이벤트 발행 코드 읽기: `backend/internal/platform/game_manager.go`, `ws/hub.go`
4. WS 이벤트 수신 코드 읽기: `frontend/src/store/gameStore.ts`
5. 불일치 목록 작성 → 수정

### 테스트 작성 요청 시

1. 기존 테스트 확인: `backend/internal/games/mafia/phases_test.go`
2. 테스트 대상 코드 읽기
3. 테스트 파일 작성 (`*_test.go`)
4. `go test ./...` 실행 및 결과 확인

### 전체 QA 실행 시

순서대로 수행:
1. **빌드 검증**: `cd backend && go build ./...`
2. **테스트 실행**: `go test ./...`
3. **경계면 검증**: DTO ↔ types.ts 교차 비교
4. **WS payload 검증**: emit 필드 ↔ onmessage 수신 필드 비교
5. **버그 리포트** 작성

## 검증 체크리스트

### HTTP API 경계면

- [ ] `dto.RoomResponse` 필드명 ↔ `types.ts Room` 필드명 일치
- [ ] `dto.PlayerDTO` 필드명 ↔ `types.ts Player` 필드명 일치
- [ ] `dto.JoinRoomResponse.PlayerID` → `res.player_id` 프론트에서 정상 저장
- [ ] `createRoom`, `joinRoom`, `joinByCode` 응답 shape 일치

### WS 이벤트 경계면

프론트 → 백엔드:
- [ ] `type: "chat"` → `dto.ActionRequest.Chat.Message` 경로 올바름
- [ ] `type: "vote"` → `dto.ActionRequest.Vote.TargetID` 경로 올바름
- [ ] `type: "kill"/"investigate"` → `dto.ActionRequest.Night.*` 경로 올바름

백엔드 → 프론트:
- [ ] `EventChat` payload의 `sender_id`, `sender_name`, `message` 필드 존재
- [ ] `EventKill` payload의 `player_id`, `role` 필드 존재
- [ ] `EventPhaseChange` payload의 `phase`, `round`, `duration`, `alive_players` 필드 존재
- [ ] `EventGameOver` payload의 `winner`, `round` 필드 존재

### RoomService 로직

- [ ] `ListPublic()` — DB 결과 중 인메모리에 없는 방 제외
- [ ] `ToRoomResponse()` — `p.IsAI == true` 플레이어 제외
- [ ] `RemovePlayer()` — HumanCount 0이 되면 방 삭제 (메모리 + DB)
- [ ] `Join()` — HumanCount >= MaxHumans 시 거부
- [ ] `GameManager.start()` — recovery path에서 AI 중복 추가 방지

## 테스트 파일 구조

새 테스트 파일은 테스트 대상 패키지 디렉토리에 `_test.go` 접미사로 생성:

```
backend/internal/platform/
├── room.go
├── room_test.go        ← 여기에 RoomService 테스트
├── handler.go
└── handler_test.go     ← 여기에 HTTP 핸들러 테스트
```

### room_test.go 뼈대

```go
package platform

import (
    "testing"
    "go.uber.org/zap"
)

func testRoomService(t *testing.T) *RoomService {
    t.Helper()
    return NewRoomService(nil, zap.NewNop())
}

func TestListPublic_OnlyInMemoryRooms(t *testing.T) {
    svc := testRoomService(t)
    // DB nil → 인메모리 fallback 경로
    // 방 2개 생성, 하나는 인메모리에서 삭제
    // ListPublic()이 메모리에 있는 방만 반환하는지 확인
}

func TestToRoomResponse_ExcludesAI(t *testing.T) {
    // AI 플레이어가 있는 방에서 ToRoomResponse() 호출
    // players 목록에 AI가 없는지 확인
}

func TestRemovePlayer_EmptyRoom(t *testing.T) {
    svc := testRoomService(t)
    // 방 생성 → 유일한 인간 플레이어 제거
    // rooms 맵에서 방이 삭제됐는지 확인
}

func TestJoin_FullRoom(t *testing.T) {
    svc := testRoomService(t)
    // MaxHumans: 1인 방 생성 (방장 1명)
    // 두 번째 참가 시도 → 에러 반환 확인
}
```

### handler_test.go 뼈대

```go
package platform

import (
    "net/http/httptest"
    "strings"
    "testing"
    
    "github.com/gofiber/fiber/v2"
    "go.uber.org/zap"
)

type mockHub struct{}
func (m *mockHub) StartGame(roomID string) error   { return nil }
func (m *mockHub) RestartGame(roomID string) error { return nil }

func setupTestApp(t *testing.T) *fiber.App {
    t.Helper()
    svc := NewRoomService(nil, zap.NewNop())
    h := NewHandler(svc, &mockHub{})
    app := fiber.New()
    h.RegisterRoutes(app)
    return app
}

func TestCreateRoom_ReturnsPlayerID(t *testing.T) {
    app := setupTestApp(t)
    body := `{"name":"테스트방","visibility":"public","max_humans":6}`
    req := httptest.NewRequest("POST", "/api/rooms", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Player-Name", "테스터")
    
    resp, _ := app.Test(req)
    // resp.StatusCode == 201
    // body에 player_id, id 필드 존재
}
```

## 실행 커맨드

```bash
# 전체 테스트
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./...

# 특정 패키지
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./internal/platform/...

# -v 상세 출력
cd /Users/yuhojin/Desktop/ai_side/backend && go test -v ./internal/platform/...

# race detector
cd /Users/yuhojin/Desktop/ai_side/backend && go test -race ./...

# 커버리지
cd /Users/yuhojin/Desktop/ai_side/backend && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
```
