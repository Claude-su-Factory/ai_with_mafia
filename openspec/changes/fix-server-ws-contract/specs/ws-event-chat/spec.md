## ADDED Requirements

### Requirement: Chat event payload structure is unified
모든 채팅 이벤트(플레이어 채팅 및 AI 채팅)는 동일한 payload 구조로 브로드캐스트되어야 한다.

이벤트 형태:
```json
{
  "type": "chat",
  "payload": {
    "sender_id": "<player_id>",
    "sender_name": "<player_name>",
    "message": "<message_text>",
    "mafia_only": false
  }
}
```

#### Scenario: Player chat uses payload wrapper
- **WHEN** 인간 플레이어가 채팅 메시지를 전송하면
- **THEN** 브로드캐스트 이벤트는 `{type: "chat", payload: {sender_id, sender_name, message, mafia_only}}` 구조를 가진다

#### Scenario: AI chat uses payload wrapper
- **WHEN** AI 플레이어가 채팅 메시지를 생성하면
- **THEN** 브로드캐스트 이벤트는 `{type: "chat", payload: {sender_id, sender_name, message, mafia_only}}` 구조를 가진다

#### Scenario: AI mafia chat is filtered to mafia only
- **WHEN** AI 마피아 플레이어가 `mafia_only: true`로 채팅하면
- **THEN** 마피아 역할의 클라이언트에게만 이벤트가 전달된다
