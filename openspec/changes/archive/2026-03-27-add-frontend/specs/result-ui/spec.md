## ADDED Requirements

### Requirement: 게임 종료 결과 화면
게임이 끝나면 결과 화면이 표시되어야 한다.

#### Scenario: 결과 화면 표시
- **WHEN** WS로 `game_over` 이벤트를 수신하면
- **THEN** 결과 화면이 오버레이 또는 화면 전환으로 표시된다
- **THEN** 승리 팀(마피아 승 / 시민 승)이 크게 표시된다
- **THEN** 모든 플레이어의 이름과 역할이 공개된다

#### Scenario: 플레이어별 결과
- **WHEN** 결과 화면이 표시되면
- **THEN** 각 플레이어에 대해 이름, 역할, 생존 여부가 표시된다

### Requirement: 게임 재시작
방장은 결과 화면에서 게임을 다시 시작할 수 있어야 한다.

#### Scenario: 방장 재시작
- **WHEN** 결과 화면에서 로컬 playerID가 방장이면
- **THEN** "다시 시작" 버튼이 표시된다
- **WHEN** 버튼을 누르면
- **THEN** `POST /api/rooms/:id/restart`가 `X-Player-ID` 헤더와 함께 전송된다
- **THEN** 대기실 화면으로 전환된다
