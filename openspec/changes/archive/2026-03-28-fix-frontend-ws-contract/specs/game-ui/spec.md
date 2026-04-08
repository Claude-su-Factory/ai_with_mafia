## MODIFIED Requirements

### Requirement: 공개 채팅
낮 토론 페이즈에서 플레이어는 채팅으로 대화할 수 있어야 한다.

#### Scenario: 채팅 메시지 전송
- **WHEN** 유저가 채팅 입력창에 메시지를 입력하고 전송하면
- **THEN** WS로 `{ type: "chat", chat: { message, mafia_only: false } }` 액션이 전송된다

#### Scenario: 채팅 메시지 수신
- **WHEN** WS로 `{ type: "chat", payload: { sender_id, sender_name, message, mafia_only } }` 이벤트를 수신하면
- **THEN** 채팅 로그에 `sender_name`과 메시지가 표시된다

#### Scenario: 마피아 전용 채팅 수신
- **WHEN** WS로 `{ type: "mafia_chat", payload: { sender_id, sender_name, message } }` 이벤트를 수신하면
- **THEN** 채팅 로그에 발신자 이름과 메시지가 마피아 전용 표시와 함께 표시된다

## ADDED Requirements

### Requirement: 게임 시작 시 게임 화면으로 전환
WaitingRoom에서 게임이 시작되면 자동으로 GameRoom으로 화면이 전환되어야 한다.

#### Scenario: phase_change 수신으로 게임 화면 전환
- **WHEN** WaitingRoom에 있는 동안 WS로 `phase_change` 이벤트를 수신하면
- **THEN** `room.status`가 `'playing'`으로 업데이트된다
- **THEN** RoomPage가 WaitingRoom 대신 GameRoom을 렌더한다

#### Scenario: initial_state로 게임 화면 직접 진입
- **WHEN** 게임이 이미 진행 중인 방에 연결하면
- **THEN** `initial_state`의 `room.status === 'playing'`으로 인해 GameRoom이 즉시 렌더된다
