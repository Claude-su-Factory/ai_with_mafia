## ADDED Requirements

### Requirement: 현재 페이즈와 타이머 표시
게임 화면은 현재 페이즈와 남은 시간을 항상 표시해야 한다.

#### Scenario: 페이즈 표시
- **WHEN** 게임이 진행 중이면
- **THEN** 현재 페이즈(낮 토론 / 투표 / 밤)가 화면 상단에 표시된다
- **THEN** 라운드 번호가 표시된다

#### Scenario: 타이머 카운트다운
- **WHEN** `initial_state` 또는 `phase_change` 이벤트를 수신하면
- **THEN** `timer_remaining_sec` 값부터 1초씩 카운트다운이 시작된다
- **WHEN** 타이머가 0이 되면
- **THEN** 타이머 표시가 멈추고 서버의 페이즈 전환을 기다린다

### Requirement: 공개 채팅
낮 토론 페이즈에서 플레이어는 채팅으로 대화할 수 있어야 한다.

#### Scenario: 채팅 메시지 전송
- **WHEN** 유저가 채팅 입력창에 메시지를 입력하고 전송하면
- **THEN** WS로 `{ type: "chat", chat: { message, mafia_only: false } }` 액션이 전송된다

#### Scenario: 채팅 메시지 수신
- **WHEN** WS로 `{ type: "chat", payload: { sender_id, sender_name, message, mafia_only } }` 이벤트를 수신하면
- **THEN** 채팅 로그에 `sender_name`과 메시지가 표시된다

### Requirement: 투표 (낮 투표 페이즈)
투표 페이즈에서 플레이어는 처형할 대상을 선택할 수 있어야 한다.

#### Scenario: 투표 대상 선택
- **WHEN** 페이즈가 `day_vote`이면
- **THEN** 생존한 다른 플레이어 목록이 투표 버튼과 함께 표시된다
- **WHEN** 플레이어를 클릭하면
- **THEN** WS로 `{ type: "vote", vote: { target_id } }` 액션이 전송된다

#### Scenario: 투표 현황 표시
- **WHEN** `vote` 이벤트를 수신하면
- **THEN** 투표 현황(누가 누구에게 투표했는지)이 실시간으로 업데이트된다

### Requirement: 마피아 야간 채팅 및 킬 선택 (밤 페이즈)
마피아 역할의 플레이어는 밤에 비공개 채널에서 대화하고 처치 대상을 선택할 수 있어야 한다.

#### Scenario: 마피아 채널 표시
- **WHEN** 페이즈가 `night`이고 `my_role`이 `mafia`이면
- **THEN** 마피아 전용 채팅 입력창과 처치 대상 선택 UI가 표시된다

#### Scenario: 마피아 전용 채팅 수신
- **WHEN** WS로 `{ type: "mafia_chat", payload: { sender_id, sender_name, message } }` 이벤트를 수신하면
- **THEN** 채팅 로그에 발신자 이름과 메시지가 마피아 전용 표시와 함께 표시된다

#### Scenario: 일반 플레이어 밤 대기
- **WHEN** 페이즈가 `night`이고 `my_role`이 `mafia`가 아니면
- **THEN** "밤입니다. 마피아가 활동 중..." 메시지만 표시된다

### Requirement: 플레이어 생존 상태 표시
사망한 플레이어는 구별되어 표시되어야 한다.

#### Scenario: 사망 처리
- **WHEN** `kill` 이벤트를 수신하면
- **THEN** 해당 플레이어가 플레이어 목록에서 사망 표시로 변경된다
- **THEN** 공개된 역할이 함께 표시된다

### Requirement: 게임 시작 시 게임 화면으로 전환
WaitingRoom에서 게임이 시작되면 자동으로 GameRoom으로 화면이 전환되어야 한다.

#### Scenario: phase_change 수신으로 게임 화면 전환
- **WHEN** WaitingRoom에 있는 동안 WS로 `phase_change` 이벤트를 수신하면
- **THEN** `room.status`가 `'playing'`으로 업데이트된다
- **THEN** RoomPage가 WaitingRoom 대신 GameRoom을 렌더한다

#### Scenario: initial_state로 게임 화면 직접 진입
- **WHEN** 게임이 이미 진행 중인 방에 연결하면
- **THEN** `initial_state`의 `room.status === 'playing'`으로 인해 GameRoom이 즉시 렌더된다
