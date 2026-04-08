## ADDED Requirements

### Requirement: WS 연결 시 initial_state 이벤트 전송
클라이언트가 WS에 연결하면 서버는 즉시 `initial_state` 타입의 이벤트를 전송해야 한다. 이 이벤트만으로 클라이언트가 현재 UI를 완전히 구성할 수 있어야 한다.

#### Scenario: 대기 중인 방에 연결
- **WHEN** 게임이 시작되지 않은 방에 WS 연결하면
- **THEN** `initial_state` 이벤트의 `room.status`가 `"waiting"`이고 `game` 필드가 `null`이다

#### Scenario: 진행 중인 게임에 연결
- **WHEN** 게임이 진행 중인 방에 WS 연결하면
- **THEN** `initial_state` 이벤트에 `room`, `game`, `my_role` 필드가 모두 포함된다
- **THEN** `game.phase`, `game.round`, `game.timer_remaining_sec`, `game.alive_player_ids`가 포함된다

#### Scenario: 재연결 시 상태 복원
- **WHEN** grace period 내에 동일 playerID로 재연결하면
- **THEN** `initial_state` 이벤트를 다시 수신하여 최신 게임 상태로 UI를 복원한다

### Requirement: 타이머 잔여 시간 제공
서버는 현재 페이즈의 남은 시간을 초 단위로 계산하여 `initial_state`에 포함해야 한다.

#### Scenario: 페이즈 시작 직후 연결
- **WHEN** 페이즈 시작 직후 클라이언트가 연결하면
- **THEN** `game.timer_remaining_sec`이 해당 페이즈 전체 duration에 가까운 값이다

#### Scenario: 페이즈 중반에 연결
- **WHEN** 120초 투표 페이즈에서 50초 경과 후 클라이언트가 연결하면
- **THEN** `game.timer_remaining_sec`이 약 70이다

#### Scenario: 타이머 만료 후 조회
- **WHEN** 페이즈 타이머가 만료된 시점에 State()를 호출하면
- **THEN** `timer_remaining_sec`이 0이다 (음수가 아님)

### Requirement: 게임 시작 시 연결된 클라이언트에게 역할 전송
게임이 시작되어 역할이 배정될 때, 이미 WS에 연결되어 있는 클라이언트에게 개별적으로 role_assigned 이벤트가 전송되어야 한다.

#### Scenario: 대기실 연결 상태에서 게임 시작
- **WHEN** 플레이어가 대기실 WS에 연결된 상태에서 방장이 게임을 시작하면
- **THEN** 해당 플레이어에게 `role_assigned` 이벤트가 개별 전송된다
- **THEN** 다른 플레이어의 역할은 전송되지 않는다

#### Scenario: role_assigned 이벤트 구조
- **WHEN** `role_assigned` 이벤트가 전송되면
- **THEN** `{ type: "role_assigned", payload: { role: "mafia"|"police"|"citizen" } }` 형식이다

### Requirement: 역할 정보는 해당 플레이어에게만 전송
`my_role` 필드는 연결한 클라이언트 본인의 역할만 포함하며, 다른 플레이어의 역할은 노출하지 않는다.

#### Scenario: 게임 진행 중 역할 수신
- **WHEN** 게임 진행 중인 방에 playerID로 연결하면
- **THEN** `initial_state.my_role`이 해당 playerID의 역할(mafia/police/citizen)이다

#### Scenario: 게임 시작 전 역할 미배정
- **WHEN** 게임 시작 전 방에 연결하면
- **THEN** `initial_state.my_role`이 빈 문자열이다
