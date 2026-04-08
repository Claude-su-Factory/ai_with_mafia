## ADDED Requirements

### Requirement: Player slot is preserved during reconnect grace period
WS 연결이 끊긴 후 grace period(기본 30초) 동안 플레이어 슬롯과 역할이 유지되어야 한다.

#### Scenario: Reconnect within grace period
- **WHEN** 플레이어의 WS 연결이 끊기고
- **WHEN** grace period 이내에 동일 playerID로 재연결하면
- **THEN** 플레이어가 게임에서 제거되지 않고 기존 역할로 세션이 복원된다

#### Scenario: Grace period expires without reconnect
- **WHEN** 플레이어의 WS 연결이 끊기고
- **WHEN** grace period가 만료될 때까지 재연결이 없으면
- **THEN** 플레이어가 게임에서 제거되고 EventPlayerReplaced가 브로드캐스트된다

#### Scenario: Grace period disabled
- **WHEN** `reconnect_grace_sec`이 0으로 설정되어 있으면
- **THEN** WS 연결 끊김 즉시 플레이어가 제거된다 (기존 동작)

### Requirement: Cross-instance reconnect is handled via Pub/Sub
다른 인스턴스에서 플레이어가 재연결할 때 원래 인스턴스의 grace timer가 취소되어야 한다.

#### Scenario: Player reconnects to different instance
- **WHEN** Instance A에서 연결이 끊기고 grace timer가 진행 중일 때
- **WHEN** 플레이어가 Instance B로 재연결하면
- **THEN** Instance B가 `player_reconnected` 이벤트를 Redis Pub/Sub에 publish한다
- **THEN** Instance A가 이벤트를 수신하여 grace timer를 취소한다
- **THEN** Instance A는 `RemovePlayer()`를 호출하지 않는다

#### Scenario: Grace timer fires before reconnect event arrives
- **WHEN** Instance A의 grace timer가 만료되어 `RemovePlayer()`가 호출되고
- **WHEN** 이후 `player_reconnected` 이벤트가 도착하면
- **THEN** 이미 제거된 상태이므로 이벤트를 무시한다 (idempotent)
