## MODIFIED Requirements

### Requirement: Room state survives instance restart
방 정보는 인스턴스가 재시작되더라도 유지되어야 한다.

#### Scenario: Room list after restart
- **WHEN** 인스턴스가 재시작되면
- **THEN** 기존 방 목록이 복구되어 GET /api/rooms에서 조회 가능하다

#### Scenario: Room join after instance change
- **WHEN** 방을 생성한 인스턴스와 다른 인스턴스로 참가 요청이 오면
- **THEN** 해당 방을 찾아 참가 처리를 완료한다

### Requirement: WebSocket messages reach all clients in a room regardless of instance
같은 방의 플레이어가 서로 다른 인스턴스에 연결되어 있어도 메시지가 모두에게 전달되어야 한다.

#### Scenario: Cross-instance broadcast
- **WHEN** Instance A의 클라이언트가 채팅을 전송하면
- **THEN** Instance B에 연결된 같은 방 클라이언트도 해당 채팅을 수신한다

#### Scenario: MafiaOnly cross-instance broadcast
- **WHEN** MafiaOnly=true 메시지가 Instance A에서 발행되면
- **THEN** Instance B에 연결된 마피아 역할 클라이언트만 수신한다

### Requirement: Game state persists across instance failures (Level 1)
인스턴스 크래시 후 다른 인스턴스가 게임을 복구할 수 있어야 한다.

#### Scenario: Game recovery after crash
- **WHEN** 게임 루프를 담당하던 인스턴스가 크래시되고
- **WHEN** 클라이언트가 재접속하면
- **THEN** 30초 이내에 다른 인스턴스가 게임을 마지막 체크포인트 페이즈부터 재시작한다
- **THEN** 페이즈 타이머는 리셋된다 (Level 1 허용)

#### Scenario: AI history recovery
- **WHEN** 게임이 복구될 때
- **THEN** AI 에이전트가 크래시 전까지의 대화 히스토리를 복원하여 맥락을 유지한다

### Requirement: Only one instance runs the game loop per room
같은 방의 게임 루프가 여러 인스턴스에서 중복 실행되어서는 안 된다.

#### Scenario: Concurrent recovery attempt
- **WHEN** 여러 인스턴스가 동시에 같은 방을 복구 시도하면
- **THEN** 하나의 인스턴스만 리더 잠금을 획득하여 게임 루프를 실행한다
- **THEN** 나머지 인스턴스는 복구를 건너뛴다
