---
name: backend-dev
description: "AI 마피아 게임 Go 백엔드 개발 스킬. 기능 구현, 버그 수정, 리팩터링, 테스트 작성을 수행한다. Go, Fiber, WebSocket, RoomService, GameManager, AI 에이전트 관련 코드 작업 시 반드시 이 스킬을 사용할 것. 다시 실행, 업데이트, 추가 수정 요청 시에도 이 스킬을 사용."
---

# Backend Dev Skill

AI 마피아 게임 Go 백엔드 작업을 위한 가이드.

## 작업 흐름

1. 요청 분석 → 영향받는 레이어 파악 (entity / platform / games / ai / repository)
2. 관련 파일 Read → 현재 구현 이해
3. 변경 구현 → Edit/Write 사용
4. 빌드 검증: `cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...`
5. 테스트 실행 (해당 시): `go test ./...`
6. 결과 보고

## 레이어별 작업 가이드

### entity 수정 시

`entity.Room`은 `sync.RWMutex`로 보호된다. 새 필드 추가 시 getter/setter 패턴을 따른다. 직접 필드 접근 대신 `GetPlayers()`, `PlayerByID()` 등 기존 메서드를 활용한다.

### RoomService 수정 시

- `ListPublic()`: DB + 인메모리 병합 시 **반드시 인메모리에 있는 방만** 반환
- `ToRoomResponse()`: AI 플레이어(`p.IsAI == true`)는 반드시 제외
- `RemovePlayer()`: `HumanCount() == 0`이면 인메모리와 DB 양쪽 삭제

### GameManager 수정 시

- `start()` 함수의 AI 추가 로직: `len(room.GetPlayers())`를 기준으로 aiCount 계산
- recovery path에서 기존 플레이어(AI 포함)가 이미 있으면 중복 추가 방지

### WS Hub 수정 시

- `ServeWS()`에서 room/player 미발견 시 반드시 `logger.Warn` 후 `c.Close()`
- rate limit 기준: 200ms per message per client

### HTTP 핸들러 수정 시

- 새 엔드포인트는 `handler.go`의 `RegisterRoutes()`에 등록
- `JoinRoomResponse` 반환 시 `PlayerID` 필드 포함 확인 (프론트가 localStorage에 저장)

## 테스트 작성 가이드

### DB 없이 RoomService 테스트

```go
func testRoomService(t *testing.T) *RoomService {
    t.Helper()
    return NewRoomService(nil, zap.NewNop())
}
```

### Fiber 핸들러 테스트

```go
func TestCreateRoom(t *testing.T) {
    svc := testRoomService(t)
    hub := &mockHub{}
    h := NewHandler(svc, hub)
    
    app := fiber.New()
    h.RegisterRoutes(app)
    
    body := `{"name":"test","visibility":"public","max_humans":6}`
    req := httptest.NewRequest("POST", "/api/rooms", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Player-Name", "테스터")
    
    resp, err := app.Test(req)
    // assertions...
}
```

### 기존 테스트 참고

`internal/games/mafia/phases_test.go`의 패턴을 참고한다:
- `newTestPlayers()` 같은 helper 함수로 test fixture 구성
- `drainEvents()` 패턴으로 채널 이벤트 수집
- Table-driven test 선호

## 자주 쓰는 커맨드

```bash
# 빌드
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...

# 전체 테스트
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./...

# 특정 패키지 테스트
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./internal/platform/...

# 특정 테스트 함수
cd /Users/yuhojin/Desktop/ai_side/backend && go test -run TestCreateRoom ./internal/platform/...

# race condition 감지
cd /Users/yuhojin/Desktop/ai_side/backend && go test -race ./...
```
