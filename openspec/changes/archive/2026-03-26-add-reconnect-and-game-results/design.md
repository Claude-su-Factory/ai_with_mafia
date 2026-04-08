## Context

현재 `hub.Unregister()`는 WS 연결 종료 즉시 `RemovePlayer()`를 호출한다. 이로 인해 네트워크 순간 끊김이나 브라우저 새로고침만으로도 플레이어가 게임에서 퇴장 처리된다. 또한 `GameResultRepository`와 DB 스키마가 이미 존재하지만 `_ = gameResultRepo`로 연결되지 않아 게임 결과가 저장되지 않는다.

이 변경은 `add-ha-and-observability` 이후에 적용된다. Redis Pub/Sub(`room:{id}` 채널)와 Redis 클라이언트 인프라가 이미 구성된 상태를 전제한다.

## Goals / Non-Goals

**Goals:**
- WS 재연결 grace period — N초 내 재연결 시 게임 슬롯과 역할 복원
- 멀티 인스턴스 환경에서 크로스 인스턴스 reconnect 감지
- 게임 종료 시 결과를 DB에 저장

**Non-Goals:**
- 플레이어 인증/identity 검증 (나중에 Google Auth로 처리 예정)
- 게임 결과 조회 API 추가
- grace period 중 해당 플레이어 대신 AI가 임시로 플레이

## Decisions

### 1. Grace Period 타이머 위치: Hub 로컬

Hub에 `pendingDisconnects map[string]*pendingDisconnect`를 추가한다. 키는 `playerID`.

```
type pendingDisconnect struct {
    timer  *time.Timer
    roomID string
    role   entity.Role
}
```

`Unregister()` 시:
- Hub rooms map에서 client 제거 (새 연결 등록 가능하도록)
- `RemovePlayer()` 호출 대신 grace timer 시작

grace timer 만료 시:
- `RemovePlayer()` 호출
- Redis에 `player_removed` 이벤트 publish

재연결(`ServeWS` 진입) 시:
- `pendingDisconnects`에 playerID가 있으면 timer 취소
- 기존 role 정보로 Client 초기화 (DB/room에서 조회)
- Redis에 `player_reconnected` 이벤트 publish

### 2. 크로스 인스턴스 Reconnect: Pub/Sub 이벤트 타입 확장

기존 `room:{id}` 채널 payload에 두 가지 내부 이벤트 타입을 추가한다.

```json
{ "type": "player_reconnected", "player_id": "...", "origin": "instance-uuid" }
{ "type": "player_removed",     "player_id": "...", "origin": "instance-uuid" }
```

`startSubscriber` goroutine에서 이 타입을 수신하면:
- `player_reconnected` → `pendingDisconnects`에서 해당 playerID timer 취소
- `player_removed` → 이미 로컬에서 처리 완료라면 무시 (idempotent)

자신이 보낸 이벤트는 `origin == instanceID` 체크로 skip (기존 WS relay와 동일).

### 3. Grace Period 기본값: 30초

`config.toml`에 `reconnect_grace_sec = 30` 추가. HA leader lock TTL(30초)과 동일하게 맞춰 일관성 유지.

설정값 0이면 grace period 비활성화 (즉시 제거, 기존 동작).

### 4. 게임 결과 저장 시점: EventGameOver 수신 즉시

`main.go`의 이벤트 포워딩 goroutine에서 `EventGameOver`를 수신할 때 `gameResultRepo.Save()`를 호출한다. 저장 실패는 로그만 남기고 게임 흐름에 영향을 주지 않는다.

`MafiaGame.endGame()` payload에 결과 데이터를 포함시켜 main.go로 전달한다:

```go
payload: map[string]any{
    "winner":       winner,
    "round":        state.Round,
    "duration_sec": int(time.Since(g.startedAt).Seconds()),
    "players":      []map[string]any{ /* id, name, role, is_ai, survived */ },
}
```

`GameResultRepository.Save()`는 이미 트랜잭션으로 구현되어 있어 그대로 활용한다.

## Risks / Trade-offs

- **Grace period 중 zombie slot**: 30초간 해당 플레이어 슬롯이 비어 있다. 투표 등 타임아웃 기반 로직은 응답 없는 플레이어를 자연스럽게 건너뜀. 문제 없음.
- **크로스 인스턴스 race**: Instance A에서 grace timer가 만료되기 직전 Instance B로 reconnect 이벤트가 오면 A가 이미 `RemovePlayer()`를 호출할 수 있음. `RemovePlayer()`는 idempotent하게 구현되어 있으므로 중복 호출은 무해함.
- **pendingDisconnects 메모리 누수**: 비정상 케이스로 timer가 fire하지 않으면 맵에 잔류. timer는 반드시 만료 시 자기 자신을 map에서 제거하도록 클로저에서 처리.

## Open Questions

없음
