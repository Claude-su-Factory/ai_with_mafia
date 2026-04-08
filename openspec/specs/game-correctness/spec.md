## ADDED Requirements

### Requirement: AI players are added to room before game starts
게임 시작 시 시스템은 `max_humans`와 총 플레이어 수(6)의 차이만큼 AI 플레이어를 생성하여 `room.Players`에 추가한 후 게임 로직을 초기화해야 한다.

#### Scenario: Game starts with one human
- **WHEN** 방에 인간 플레이어 1명이 있고 max_humans=1로 게임이 시작되면
- **THEN** 시스템은 AI 플레이어 5명을 room.Players에 추가하고 총 6명으로 게임을 시작한다

#### Scenario: AI agents receive correct player list
- **WHEN** AI 에이전트가 spawn될 때
- **THEN** room.Players에는 이미 AI 플레이어가 포함되어 있어야 한다

### Requirement: Human player actions are routed to game logic
인간 플레이어가 WebSocket으로 전송하는 채팅, 투표, 밤 행동은 반드시 활성 게임의 `HandleAction`으로 전달되어야 한다.

#### Scenario: Human sends chat message
- **WHEN** 인간 플레이어가 `{"type":"chat","chat":{"message":"..."}}` 메시지를 전송하면
- **THEN** 해당 방의 모든 플레이어에게 채팅이 브로드캐스트되고 AI 에이전트 히스토리에 기록된다

#### Scenario: Human votes
- **WHEN** 인간 플레이어가 `{"type":"vote","vote":{"target_id":"..."}}` 메시지를 전송하면
- **THEN** 투표가 게임 state에 기록되고 현황이 브로드캐스트된다

#### Scenario: Human sends night kill action
- **WHEN** 마피아 인간 플레이어가 밤 페이즈에 `{"type":"kill","night":{"action_type":"kill","target_id":"..."}}` 메시지를 전송하면
- **THEN** 해당 밤 행동이 게임 state에 기록된다

### Requirement: AI mafia decides kill target at night
마피아 역할의 AI 에이전트는 밤 페이즈가 시작되면 처치 대상을 결정하여 게임에 제출해야 한다.

#### Scenario: AI mafia submits kill decision
- **WHEN** 밤 페이즈가 시작되고 AI 에이전트의 역할이 마피아이면
- **THEN** AI는 살아있는 비마피아 플레이어 중 한 명을 선택하여 kill 액션을 제출한다

### Requirement: Chat events include sender name
게임에서 발생하는 chat 이벤트 payload에는 발언자의 이름(`sender_name`)이 포함되어야 한다.

#### Scenario: Chat payload includes sender name
- **WHEN** 플레이어가 채팅 메시지를 전송하면
- **THEN** 브로드캐스트되는 이벤트 payload에 `sender_name` 필드가 포함된다

### Requirement: Concurrent access to game state is thread-safe
게임 상태와 방 플레이어 목록에 대한 동시 접근은 mutex로 보호되어야 한다.

#### Scenario: Multiple goroutines access activeGames map
- **WHEN** StartGame과 NotifyEvent가 동시에 호출되면
- **THEN** race condition 없이 안전하게 처리된다

#### Scenario: room.Players is read during game
- **WHEN** 게임 로직이 room.Players 슬라이스를 읽을 때
- **THEN** Room의 mutex를 통해 보호된 접근을 사용한다

### Requirement: Game goroutines respect server context
게임 goroutine은 서버 shutdown signal을 전파받은 context를 사용하여 서버 종료 시 정상적으로 종료되어야 한다.

#### Scenario: Server shuts down during active game
- **WHEN** 서버가 SIGINT/SIGTERM을 받아 종료될 때
- **THEN** 진행 중인 게임 goroutine이 server context 취소를 통해 종료된다

### Requirement: Game engine events are broadcast to WebSocket clients
`game.Subscribe()`에서 발생하는 모든 게임 엔진 이벤트(페이즈 변경, kill, 투표 결과, 게임 오버)는 해당 방의 인간 WebSocket 클라이언트에 전달되어야 한다.

#### Scenario: Phase change reaches human clients
- **WHEN** 게임 엔진이 페이즈 변경 이벤트를 발행하면
- **THEN** 해당 방에 연결된 모든 인간 WS 클라이언트가 phase_change 메시지를 수신한다

#### Scenario: Kill result reaches human clients
- **WHEN** 플레이어가 처형되면
- **THEN** 해당 방의 모든 WS 클라이언트가 kill 이벤트를 수신한다

#### Scenario: Mafia-only events are filtered
- **WHEN** MafiaOnly=true인 게임 이벤트가 발행되면
- **THEN** 마피아 역할의 WS 클라이언트에게만 전달된다

### Requirement: WebSocket client role is synchronized after game start
WS 클라이언트에 저장된 역할 정보는 게임 시작 후 역할 배정 완료 즉시 동기화되어야 한다.

#### Scenario: Mafia client receives mafia-only broadcast
- **WHEN** 게임이 시작되어 역할이 배정되면
- **THEN** 마피아 역할의 인간 플레이어 WS 클라이언트의 Role 필드가 업데이트된다

#### Scenario: Client role filter is accurate during night phase
- **WHEN** 밤 페이즈에 MafiaOnly=true 브로드캐스트가 발생하면
- **THEN** 마피아 인간 플레이어는 해당 메시지를 수신하고, 시민/경찰 인간 플레이어는 수신하지 않는다
