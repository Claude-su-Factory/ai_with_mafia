## ADDED Requirements

### Requirement: 독립 AI Agent goroutine
각 AI 플레이어는 독립 goroutine에서 실행되며, 자신만의 Claude API 세션(대화 히스토리)을 유지해야 한다.

#### Scenario: AI Agent 시작
- **WHEN** 게임이 시작되면
- **THEN** AI 플레이어 수만큼 독립 goroutine이 생성되고, 각자 자신의 역할 system prompt로 초기화되어야 한다

#### Scenario: 방 종료 시 Agent 정리
- **WHEN** 방이 종료되거나 게임이 끝나면
- **THEN** 모든 AI Agent goroutine이 context cancellation을 통해 정상 종료되어야 한다

---

### Requirement: 역할별 정보 격리
각 AI Agent는 자신의 역할에 맞는 정보만 알아야 한다. 다른 AI의 역할을 알 수 없어야 한다.

#### Scenario: 마피아 AI의 공범 인지
- **WHEN** 마피아 역할의 AI가 생성되면
- **THEN** system prompt에 공범 목록이 포함되어야 하고, 공범 이외의 플레이어 역할은 포함되지 않아야 한다

#### Scenario: 시민 AI의 역할 제한
- **WHEN** 시민 역할의 AI가 생성되면
- **THEN** system prompt에는 자신이 시민임만 포함되고, 다른 플레이어의 역할 정보는 포함되지 않아야 한다

#### Scenario: 경찰 AI의 조사 결과 수신
- **WHEN** 경찰 AI가 조사를 완료하면
- **THEN** 조사 결과가 해당 AI의 private context에만 추가되고, 다른 AI에게는 전달되지 않아야 한다

---

### Requirement: 동시 Claude API 호출 제어
여러 AI Agent가 동시에 Claude API를 호출할 때 rate limit 초과를 방지하기 위해 동시 호출 수를 제한해야 한다.

#### Scenario: 동시 호출 제한 준수
- **WHEN** 설정된 max_concurrent 수를 초과하는 AI가 동시에 응답하려 하면
- **THEN** 초과분은 대기하다가 슬롯이 확보되면 순차적으로 호출되어야 한다

---

### Requirement: 대화 히스토리 크기 제한
AI Agent의 대화 히스토리가 무한히 증가하여 Claude API token limit을 초과하지 않도록 제한해야 한다.

#### Scenario: 히스토리 초과 시 오래된 메시지 제거
- **WHEN** 대화 히스토리가 history_max 설정값을 초과하면
- **THEN** system prompt는 유지하되 가장 오래된 메시지부터 제거되어야 한다

---

### Requirement: 자연스러운 응답 타이밍
AI Agent는 즉시 응답하지 않고, 사람처럼 보이도록 설정된 범위 내에서 랜덤 딜레이 후 응답해야 한다.

#### Scenario: 응답 딜레이 적용
- **WHEN** AI Agent가 응답을 생성하면
- **THEN** response_delay_min ~ response_delay_max 사이의 랜덤 딜레이 후 메시지가 전송되어야 한다

#### Scenario: 동시 다발 응답 방지
- **WHEN** 여러 AI Agent가 같은 이벤트에 반응할 때
- **THEN** 동시에 메시지를 보내지 않고 순차적으로 발언해야 한다

---

### Requirement: 모델 설정 분리
Claude API 호출 시 사용할 모델은 config에서 관리되어야 하며, 코드 변경 없이 전환 가능해야 한다.

#### Scenario: 일반 채팅 응답 모델
- **WHEN** AI Agent가 채팅 메시지에 응답할 때
- **THEN** config의 ai.model_default 모델을 사용해야 한다

#### Scenario: 중요 판단 모델
- **WHEN** AI Agent가 투표 또는 밤 행동을 결정할 때
- **THEN** config의 ai.model_reasoning 모델을 사용해야 한다
