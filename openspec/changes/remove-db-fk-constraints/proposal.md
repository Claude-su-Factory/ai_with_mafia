## Why

테스트 단계에서 DB 테이블을 자유롭게 삭제하거나 재구성할 때 FK 제약조건이 삭제 순서를 강제하여 반복적인 불편함을 야기한다. 또한 `game_results`는 현재 코드베이스에서 write-only(INSERT만 존재, SELECT/JOIN 없음)이므로 DB 레벨 정합성 보장보다 애플리케이션 레벨 관리로 충분하다.

## What Changes

- `game_results.room_id → rooms(id)` FK 제약조건 제거
- `game_result_players.game_result_id → game_results(id)` FK 제약조건 제거
- 기존 데이터 보존을 위해 새 마이그레이션(`000005`) 추가 — `ALTER TABLE ... DROP CONSTRAINT`
- `000002_create_game_results.up.sql`에서 `REFERENCES` 구문 제거 — 신규 환경에서도 FK 없이 생성되도록

## Capabilities

### New Capabilities

없음.

### Modified Capabilities

- `platform-core`: DB 스키마에서 FK 제약조건을 제거하고, 테이블 간 참조 정합성을 애플리케이션 레벨에서 관리한다는 요구사항 추가

## Impact

- `backend/migrations/000002_create_game_results.up.sql`: `REFERENCES` 절 제거
- `backend/migrations/000005_remove_fk_constraints.up.sql`: 신규 — 기존 DB의 FK 제약 제거 DDL
- `backend/migrations/000005_remove_fk_constraints.down.sql`: 신규 — 롤백용 FK 복원 DDL
- 애플리케이션 코드 변경 없음 (이미 정합성을 앱 레벨에서 처리 중)
