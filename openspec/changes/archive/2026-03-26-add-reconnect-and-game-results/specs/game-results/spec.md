## ADDED Requirements

### Requirement: Game result is persisted when a game ends
게임이 종료될 때 결과가 DB에 저장되어야 한다.

#### Scenario: Game ends with a winner
- **WHEN** 마피아 게임에서 승자가 결정되면
- **THEN** winner_team, round_count, duration_sec이 `game_results` 테이블에 저장된다
- **THEN** 각 플레이어의 player_id, player_name, role, is_ai, survived 정보가 `game_result_players` 테이블에 저장된다

#### Scenario: Result save failure does not stop the game
- **WHEN** `gameResultRepo.Save()`가 에러를 반환하면
- **THEN** 에러가 `logger.Error()`로 기록된다
- **THEN** 게임 종료 흐름은 정상적으로 계속된다

### Requirement: Game duration is accurately tracked
게임 시작부터 종료까지의 소요 시간이 초 단위로 기록되어야 한다.

#### Scenario: Duration calculated from game start
- **WHEN** 게임이 시작되면 시작 시각이 기록되고
- **WHEN** 게임이 종료되면
- **THEN** `duration_sec`이 종료 시각 - 시작 시각으로 계산되어 저장된다
