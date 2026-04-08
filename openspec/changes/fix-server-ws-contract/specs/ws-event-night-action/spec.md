## ADDED Requirements

### Requirement: night_action investigation result is sent only to the police player
경찰 조사 결과 `night_action` 이벤트는 조사를 수행한 경찰 플레이어에게만 전송되어야 한다. 다른 플레이어(마피아 포함)에게는 전달되지 않아야 한다.

이벤트 형태:
```json
{
  "type": "night_action",
  "payload": {
    "result": "마피아" | "시민",
    "target_name": "<조사한 플레이어 이름>"
  }
}
```

#### Scenario: Investigation result reaches police player
- **WHEN** 경찰 플레이어가 밤에 조사 액션을 제출하면
- **THEN** 해당 경찰 플레이어의 WS 클라이언트에 `night_action` 이벤트가 전송된다

#### Scenario: Investigation result does not reach mafia players
- **WHEN** 경찰 플레이어가 밤에 조사 액션을 제출하면
- **THEN** 마피아 역할의 WS 클라이언트는 `night_action` 이벤트를 수신하지 않는다

#### Scenario: Investigation result does not reach citizen players
- **WHEN** 경찰 플레이어가 밤에 조사 액션을 제출하면
- **THEN** 시민 역할의 WS 클라이언트는 `night_action` 이벤트를 수신하지 않는다
