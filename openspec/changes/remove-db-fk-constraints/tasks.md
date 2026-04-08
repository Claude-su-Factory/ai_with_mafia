## 1. 기존 마이그레이션 수정

- [x] 1.1 `000002_create_game_results.up.sql`에서 `game_results` 테이블의 `room_id` 컬럼에 있는 `REFERENCES rooms(id) ON DELETE CASCADE` 절 제거
- [x] 1.2 `000002_create_game_results.up.sql`에서 `game_result_players` 테이블의 `game_result_id` 컬럼에 있는 `REFERENCES game_results(id) ON DELETE CASCADE` 절 제거

## 2. 새 마이그레이션 추가

- [x] 2.1 `000005_remove_fk_constraints.up.sql` 생성 — `ALTER TABLE game_results DROP CONSTRAINT IF EXISTS game_results_room_id_fkey`
- [x] 2.2 `000005_remove_fk_constraints.up.sql`에 `ALTER TABLE game_result_players DROP CONSTRAINT IF EXISTS game_result_players_game_result_id_fkey` 추가
- [x] 2.3 `000005_remove_fk_constraints.down.sql` 생성 — FK 복원 DDL (`ALTER TABLE game_results ADD CONSTRAINT ...`, `ALTER TABLE game_result_players ADD CONSTRAINT ...`)

## 3. 검증

- [x] 3.1 `go build ./...` 실행하여 빌드 오류 없음 확인
