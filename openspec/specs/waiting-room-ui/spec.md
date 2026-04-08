## ADDED Requirements

### Requirement: 대기실 플레이어 목록 표시
대기실은 현재 방에 참가한 플레이어 목록을 보여야 한다.

#### Scenario: 플레이어 목록 표시
- **WHEN** 유저가 `/rooms/:id`에 접속하고 방 상태가 `waiting`이면
- **THEN** 참가한 플레이어 목록(이름, AI 여부)이 표시된다
- **THEN** 방장 플레이어에 표시가 나타난다

#### Scenario: 비공개 방 코드 표시
- **WHEN** 방의 `visibility`가 `private`이면
- **THEN** 참가 코드가 화면에 표시된다

### Requirement: 방장 게임 시작
방장만 게임 시작 버튼을 볼 수 있고 누를 수 있어야 한다.

#### Scenario: 방장 게임 시작
- **WHEN** 로컬 playerID가 방의 `host_id`와 일치하면
- **THEN** "게임 시작" 버튼이 표시된다
- **WHEN** 버튼을 누르면
- **THEN** `POST /api/rooms/:id/start`가 `X-Player-ID` 헤더와 함께 전송된다

#### Scenario: 일반 플레이어 대기
- **WHEN** 로컬 playerID가 방장이 아니면
- **THEN** "게임 시작" 버튼이 표시되지 않는다
- **THEN** "방장이 게임을 시작하기를 기다리는 중..." 메시지가 표시된다

### Requirement: 게임 시작 시 자동 이동
WS를 통해 게임 시작 이벤트를 수신하면 게임 화면으로 전환해야 한다.

#### Scenario: phase_change 수신으로 전환
- **WHEN** WS로 `phase_change` 이벤트를 수신하면
- **THEN** 같은 `/rooms/:id` URL에서 게임 화면 컴포넌트로 자동 전환된다
