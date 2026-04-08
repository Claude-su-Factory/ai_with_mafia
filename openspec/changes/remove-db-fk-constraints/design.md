## Context

현재 `game_results`와 `game_result_players` 테이블은 각각 `rooms`, `game_results` 테이블에 대한 FK 제약을 보유하고 있다. 반면 `game_states`, `ai_histories`는 이미 FK 없이 운영 중이다.

코드 분석 결과 `game_results`는 게임 종료 시 `GameResultRepository.Save()`를 통한 INSERT만 수행하며, `room_id`를 통해 `rooms`를 역참조하는 SELECT/JOIN은 존재하지 않는다. 즉 DB FK가 보장하는 참조 무결성이 현재 앱 동작에 기여하지 않는다.

현재 DB에 데이터가 존재하므로 테이블 재생성 대신 `ALTER TABLE ... DROP CONSTRAINT` 방식으로 처리한다.

## Goals / Non-Goals

**Goals:**
- `game_results`, `game_result_players`의 FK 제약 제거
- 기존 데이터 보존
- 신규 환경에서도 FK 없이 테이블이 생성되도록 `000002` 마이그레이션 수정
- 롤백 가능한 마이그레이션 구조 유지

**Non-Goals:**
- 애플리케이션 코드 변경 (정합성은 이미 앱 레벨에서 관리 중)
- `rooms`, `game_states`, `ai_histories` 등 다른 테이블 변경
- 데이터 정합성 검증 로직 추가

## Decisions

### 새 마이그레이션 추가 vs 기존 마이그레이션 수정

**결정:** 두 가지 모두 적용 — 기존 `000002` 수정 + 새 `000005` 추가

- **`000002` 수정**: 신규 환경(첫 마이그레이션 실행)에서 FK 없이 테이블이 생성되도록. 마이그레이션 파일은 최종 의도된 스키마를 반영해야 한다.
- **`000005` 추가**: 기존 DB에 이미 적용된 FK를 제거. `golang-migrate`는 이미 실행된 마이그레이션을 재실행하지 않으므로 별도 파일 필수.

### FK 제약 이름

PostgreSQL 자동 생성 이름을 사용한다:
- `game_results_room_id_fkey`
- `game_result_players_game_result_id_fkey`

`IF EXISTS` 절을 사용해 제약이 이미 없는 환경(신규 환경)에서도 안전하게 실행되도록 한다.

## Risks / Trade-offs

- **고아 데이터 발생 가능**: room 삭제 후 `game_results`가 남을 수 있음 → 현재 앱이 `game_results`를 조회하지 않으므로 실질적 영향 없음. 향후 조회 기능 추가 시 앱 레벨에서 처리.
- **CASCADE 자동 삭제 소멸**: room 삭제 시 관련 `game_results`가 자동 삭제되지 않음 → `game_results`는 히스토리 기록 목적이므로 보존이 오히려 자연스러움.
- **롤백 복잡성**: `000005.down.sql`로 FK 복원 가능하나, 롤백 시점에 고아 데이터가 존재하면 FK 복원이 실패할 수 있음 → 테스트 단계에서 데이터 정합성이 보장되지 않는 상황 자체가 이 변경의 전제이므로 허용.

## Migration Plan

1. `000002_create_game_results.up.sql`에서 `REFERENCES` 절 제거
2. `000005_remove_fk_constraints.up.sql` 생성 — `ALTER TABLE ... DROP CONSTRAINT IF EXISTS`
3. `000005_remove_fk_constraints.down.sql` 생성 — FK 복원 DDL
4. 백엔드 서버 재시작 → `golang-migrate`가 `000005` 자동 적용
5. 빌드 확인: `go build ./...`
