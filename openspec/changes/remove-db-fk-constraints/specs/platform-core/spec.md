## MODIFIED Requirements

### Requirement: DB 스키마 관리
DB 스키마는 `golang-migrate`를 통해 순차적으로 관리된다. 테이블 간 참조 정합성은 애플리케이션 레벨에서 관리하며, DB 레벨 FK 제약조건을 사용하지 않는다.

#### Scenario: 신규 환경 마이그레이션
- **WHEN** 새로운 환경에서 마이그레이션을 처음 실행하면
- **THEN** `game_results`, `game_result_players` 테이블이 FK 제약조건 없이 생성되어야 한다

#### Scenario: 기존 환경 FK 제거
- **WHEN** 기존 DB에 마이그레이션 `000005`를 적용하면
- **THEN** `game_results_room_id_fkey`와 `game_result_players_game_result_id_fkey` 제약조건이 제거되어야 한다

#### Scenario: 이미 FK가 없는 환경에서의 마이그레이션
- **WHEN** FK 제약조건이 이미 없는 환경에서 `000005`를 실행하면
- **THEN** `IF EXISTS` 절로 인해 오류 없이 실행이 완료되어야 한다

#### Scenario: 마이그레이션 롤백
- **WHEN** `000005.down.sql`로 롤백하면
- **THEN** FK 제약조건이 복원되어야 한다 (단, 고아 데이터 존재 시 복원이 실패할 수 있다)

#### Scenario: 임의 순서 DROP TABLE
- **WHEN** 테스트 환경에서 테이블을 임의 순서로 삭제하면
- **THEN** FK 제약조건이 없으므로 순서 강제 없이 삭제가 완료되어야 한다
