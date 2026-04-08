## ADDED Requirements

### Requirement: Critical errors are always logged
게임 로직 실행 중 발생하는 에러는 반드시 로깅되어야 한다.

#### Scenario: HandleAction error is logged
- **WHEN** `game.HandleAction()`이 에러를 반환하면
- **THEN** `logger.Error()`로 roomID, playerID, action type과 함께 기록된다

#### Scenario: Game loop goroutine error is logged
- **WHEN** goroutine 내부에서 패닉이나 에러가 발생하면
- **THEN** 해당 에러가 로깅되고 goroutine이 안전하게 종료된다

### Requirement: Event channel drops are logged
이벤트 채널이 가득 차 이벤트가 드롭될 때 로깅되어야 한다.

#### Scenario: Agent event channel full
- **WHEN** AI 에이전트의 eventCh가 가득 차 이벤트가 드롭되면
- **THEN** `logger.Warn()`으로 드롭된 이벤트 타입과 playerID가 기록된다

#### Scenario: Game event channel full
- **WHEN** mafia game의 eventCh가 가득 차 이벤트가 드롭되면
- **THEN** `logger.Warn()`으로 드롭된 이벤트와 roomID가 기록된다

#### Scenario: WS broadcast channel full
- **WHEN** WS 클라이언트의 send 채널이 가득 차 메시지가 드롭되면
- **THEN** `logger.Warn()`으로 playerID와 roomID가 기록된다

### Requirement: WebSocket write errors are logged
WS 쓰기 실패는 클라이언트 상태 불일치를 유발하므로 로깅되어야 한다.

#### Scenario: Initial state send fails
- **WHEN** WS 연결 직후 초기 상태 메시지 전송이 실패하면
- **THEN** `logger.Warn()`으로 기록되고 연결이 정리된다

### Requirement: JSON marshal/unmarshal errors are logged
직렬화 에러는 데이터 손상으로 이어질 수 있으므로 로깅되어야 한다.

#### Scenario: Broadcast payload marshal fails
- **WHEN** `hub.Broadcast()`에서 JSON marshal이 실패하면
- **THEN** `logger.Error()`로 기록되고 해당 메시지 전송을 건너뛴다

### Requirement: Secure random ID generation uses error-checked crypto/rand
방 ID 및 참가 코드 생성 시 crypto/rand 에러가 처리되어야 한다.

#### Scenario: rand.Read fails
- **WHEN** `crypto/rand.Read()`가 실패하면
- **THEN** 에러를 상위로 전파하여 방 생성이 실패로 처리된다
