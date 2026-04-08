## ADDED Requirements

### Requirement: phase_change event includes round number
모든 `phase_change` 이벤트 payload에는 현재 게임 라운드 번호가 포함되어야 한다.

이벤트 형태:
```json
{
  "type": "phase_change",
  "payload": {
    "phase": "<day_discussion|day_vote|night>",
    "round": 2,
    "timer_sec": 60
  }
}
```

#### Scenario: Day discussion phase_change includes round
- **WHEN** 낮 토론 페이즈가 시작되면
- **THEN** phase_change payload에 `round` 필드가 포함된다

#### Scenario: Day vote phase_change includes round
- **WHEN** 낮 투표 페이즈가 시작되면
- **THEN** phase_change payload에 `round` 필드가 포함된다

#### Scenario: Night phase_change includes round
- **WHEN** 밤 페이즈가 시작되면
- **THEN** phase_change payload에 `round` 필드가 포함된다

#### Scenario: Round increments correctly across phases
- **WHEN** 라운드가 2인 상태에서 phase_change가 발생하면
- **THEN** payload의 `round` 값이 2이다
