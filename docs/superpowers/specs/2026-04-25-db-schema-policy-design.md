# DB 스키마 관리 정책 — 자동 Migration 폐기 + FK 금지

**Status:** Retroactive spec — 결정과 구현이 먼저 (`d43c61d`) 되었고, 본 문서는 워크플로우 위반을 만회하기 위해 사후 작성됨. 자체 검토 사이클을 돌려 누락된 결함을 발굴하는 것이 본 spec 의 실질 가치.

**Related:**
- ARCHITECTURE.md §4.14 — 결정 로그
- CLAUDE.md "DB 스키마 정책 (MANDATORY)" — 영구 규칙 4가지
- README.md "DB Schema" — 단일 진실 위치

---

## 1. Context & Goal

이전 상태:
- `golang-migrate` 가 backend 부팅 시 `m.Up()` 자동 실행
- `backend/migrations/` 의 7쌍 SQL 적용
- 스키마 변경 = 새 migration 파일 작성 + 부팅 시 자동 적용

사용자 결정 (2026-04-25):
- 자동 migration 도구 폐기 — 운영상 불편
- FK 금지 — 성능·분산 우려

목표: **단순한 운영 모델로 전환**. README 기반 수동 적용 + 애플리케이션 레이어 무결성.

### Non-goals (이 결정 범위 밖)

- 기존 운영 DB 의 데이터 백필 (현재 운영 환경 없음 — 영향 없음)
- 별도 migration runner job 도입 — 미래 트래픽 증가 시 재평가
- 테이블 구조 자체 변경 (FK 제거는 이미 `000005` 에서 적용된 상태였음 → 추가 ALTER 불필요)

---

## 2. Decision

### 2.1 자동 migration 폐기

| 항목 | 변경 전 | 변경 후 |
|------|--------|--------|
| 부팅 시 동작 | `RunMigrations` 호출 → `golang-migrate.Up()` | 없음 (DB 미스 시 startup 실패하지 않음, 첫 query 시 에러) |
| 스키마 변경 절차 | 새 migration 파일 + 부팅 | README DDL 갱신 + 사람이 `psql` 적용 |
| 단일 진실 위치 | `backend/migrations/00000N_*.up.sql` (8개 파일) | `README.md` "DB Schema" 섹션 |
| 의존성 | `golang-migrate/migrate/v4`, `jackc/pgerrcode` | 제거 |

### 2.2 FK 금지

- `FOREIGN KEY` / `REFERENCES` 절을 어떤 신규/기존 테이블에도 추가하지 않는다
- 기존 FK 는 `000005_remove_fk_constraints` 에서 이미 모두 제거된 상태였음 (역사적 사실)
- 무결성은 애플리케이션 레이어가 담당 (트랜잭션, `ON CONFLICT`, `INSERT ... SELECT WHERE EXISTS`)

### 2.3 Why (4축 렌즈와 정합)

| 축 | 영향 |
|---|------|
| ① 비용 | 작은 positive — migration runtime 제거, FK lock contention 제거 |
| ② 수익 | 직접 영향 없음 |
| ③ 리텐션 | 운영자(개발자) 마찰 감소 → 간접 positive |
| ④ 인간 밀도 | 영향 없음 |

CLAUDE.md "Unit Economics 4축 중 최소 둘에 positive" 통과: ①③.

---

## 3. Implementation Detail

### 3.1 Code 변경

- `backend/internal/repository/db.go` → `RunMigrations` 함수 + `golang-migrate` 임포트 제거. `NewPool` 만 남김
- `backend/cmd/server/main.go` → 부팅 시 migration 호출 블록 제거. 포인터 주석으로 대체
- `backend/migrations/` 디렉토리 삭제 (14 파일)
- `go mod tidy` → indirect deps 정리

### 3.2 README.md 구조

```
README.md
├── 프로젝트 개요
├── 빠른 시작 (docker-compose + DDL 적용 + backend/frontend 실행)
├── DB Schema
│   ├── 1) rooms
│   ├── 2) game_results + game_result_players
│   ├── 3) game_states
│   ├── 4) ai_histories
│   ├── 5) users
│   └── 6) game_metrics
├── 스키마 변경 절차
└── 디렉토리 구조
```

모든 `CREATE TABLE` / `CREATE INDEX` 는 `IF NOT EXISTS` — 재실행 안전.

### 3.3 영구 규칙 등록 위치

- **CLAUDE.md** — `## DB 스키마 정책 (MANDATORY)` 섹션. 4가지 규칙 + 이유 요약
- **ARCHITECTURE.md §4.14** — 결정 로그 (Why/How/trade-off)
- **README.md** — DDL 본문 + "스키마 변경 절차" 섹션

---

## 4. Concurrency & Distribution Analysis

CLAUDE.md "동시성·분산 안전성" 렌즈에 따라 4가지 질문에 답한다.

### 4.1 자동 migration 제거의 분산 영향

| 질문 | 답 |
|------|-----|
| 상태 위치 | DB 스키마 자체 (Postgres 서버 측). 코드는 무관 |
| Cross-Pod 일관성 | 모든 Pod 가 동일 DB 를 보므로 자동으로 일관 |
| Eventual consistency 경계 | 없음. 스키마는 사람이 한 번 적용하면 즉시 모든 Pod 가 같은 view |
| 멀티 Pod 이관 시 | **개선됨**. 변경 전엔 N Pod 가 동시 부팅하면 `golang-migrate` 의 advisory lock 으로 직렬화되긴 했지만 lock 대기·실패 시 정합성 위험. 변경 후엔 사람이 `psql` 한 번 → 모든 Pod 가 새 스키마로 즉시 진입 |

### 4.2 FK 제거의 무결성 비용 — 어디서 보장?

FK 가 제공하던 보장을 누가 대신 책임지는지 매핑한다:

| 무결성 시나리오 | FK 존재 시 | FK 없는 현재 보장 위치 | 위험 평가 |
|------|--------|--------|--------|
| `game_result_players.game_result_id` → `game_results.id` 존재 | DB FK | `GameResultRepository.Save` 가 단일 트랜잭션에서 부모 INSERT 후 자식 INSERT — 부모 실패 시 자식도 미저장. `Save` 외부에서 자식 INSERT 하지 않음 (코드베이스 grep 으로 확인 가능) | 낮음 — 단일 진입점 |
| 자식 INSERT 시점에 부모 row 존재 | DB FK | 위 트랜잭션이 같은 connection 에서 순서대로 실행 → race 없음 | 낮음 |
| 부모 DELETE 시 cascade | DB FK CASCADE | **현재 코드는 `game_results` DELETE 안 함** (관측·통계 데이터, retention 정책 부재). retention 도입 시 별도 작업으로 자식 row 정리 필요 — TODO 로 ROADMAP 등록 권장 | **중간 — 미래 위험** |
| 부모 UPDATE id 시 cascade | DB FK ON UPDATE | `game_results.id` 는 PK + UUID — 갱신 안 함. N/A | 없음 |
| 외부 도구가 raw SQL 로 자식 INSERT | DB FK 가 차단 | 차단 안 됨. 운영자가 실수로 고아 row 생성 가능 | 낮음 — 외부 도구 진입점 부재 |

### 4.3 발견된 GAP — retention 정책 부재

`game_results` / `game_result_players` 는 cascade delete 의 사용자가 없다 (현재 코드 기준). 하지만 미래에 데이터 retention 정책 도입 시 (예: "1년 이상 된 레코드 삭제") FK CASCADE 가 자동으로 처리하던 부분을 **애플리케이션이 자식 → 부모 순서로 명시 삭제** 해야 한다. 이 작업이 잊혀지면 `game_result_players` 가 점진적으로 orphan row 누적.

→ **ROADMAP 에 retention 작업의 사전 조건으로 "삭제 순서 가이드" 등록 필요** (이번 자체 검토에서 발견된 결함).

### 4.4 자동 migration 의 분산 race 회피 효과

이전 모델에서는 N Pod 가 동시 부팅 시 `golang-migrate` 의 internal lock 으로 직렬화 — 하지만 lock 획득 실패 시 부팅 실패. 이 race 가 production 에서 hot deploy 중 잠깐 모든 Pod 가 unhealthy 가 되는 시나리오를 만들 수 있었다. 변경 후엔 스키마 변경이 코드 배포와 분리 → **롤링 deploy 안전성 향상**.

---

## 5. Failure Modes & Error Handling

### 5.1 사람이 README DDL 적용을 잊고 backend 기동

- **현상:** 첫 SQL query 시 `relation "X" does not exist` 에러
- **영향:** 해당 query 가 속한 핸들러만 500. 다른 경로는 정상 (예: 광고 metric 만 깨짐 — fail-open 정책으로 로그만 남고 게임 진행)
- **검출:** `pgxpool.Pool.Ping(ctx)` 는 단순 connectivity 만 확인 → 스키마 검증 안 함
- **개선 옵션 (여기 spec 범위 밖):** startup 시 sentinel SELECT 로 스키마 존재 검증 (예: `SELECT 1 FROM rooms LIMIT 1`). 미적용 — README 가이드로 충분 가정. 향후 실제로 사고가 나면 도입

### 5.2 README DDL 과 운영 DB 가 drift

- **위험:** 사람이 운영 DB 에만 컬럼을 추가하고 README 갱신 누락 → 다음 deploy 가 새 컬럼을 모름
- **방지:** PR 리뷰 시 README 갱신 여부 체크. CLAUDE.md 의 "스키마 변경 시 PR 안에서 README 갱신" 규칙으로 강제
- **검출 자동화:** 향후 CI 에서 `pg_dump --schema-only` ↔ README DDL 비교 가능 (Tier 3 작업)

### 5.3 PR 에서 SQL 만 변경하고 코드는 변경 안 함

- 스키마 추가는 무해, 제거는 코드가 따라가야 함
- README 가 단일 진실이라 PR 작성자가 schema diff 를 시각적으로 확인 가능

### 5.4 FK 금지 정책의 우회 경로

- 외부 ORM·migration 도구가 자동으로 FK 를 추가할 위험 — 현재 ORM 미사용 (`pgx` 직접 사용) → N/A
- 미래 ORM 도입 시 FK 옵션을 명시적으로 끄도록 코드 리뷰 단계에서 차단

---

## 6. Testing Strategy

### 6.1 자동 검증

- `go build ./...` → migration 모듈 제거 후 빌드 성공 (확인 완료, `d43c61d`)
- `go test ./...` → 기존 테스트 영향 없음 (`internal/repository` 테스트는 nil-pool 경로만 사용 → 자동 migration 무관)

### 6.2 수동 검증 (Phase A 검증 runbook 시점)

- README DDL 을 빈 Postgres 에 적용 → 7개 테이블 + 인덱스 생성 확인
- Backend 기동 → 첫 게임 플레이 → `game_results` / `game_metrics` row 정상 INSERT
- `psql -c "\d+ game_metrics"` 로 FK 0건 확인

### 6.3 회귀 방지 테스트 (선택)

- README DDL 을 별도 `_workspace/schema.sql` 같은 파일로 export 하고, CI 에서 `psql --dry-run` 같은 syntax 검증
- 현재 미적용. 우선순위 낮음

---

## 7. 자체 검토 사이클 (Spec Self-Review)

### 7.1 Placeholder scan

- TBD/TODO: 0건
- 정량 기준 미명시: 의도적. 정책 spec 이라 measurement 가 spec 외부 (실측은 운영 모니터링 영역)

### 7.2 Internal consistency

- §2.1, §3, README 의 "테이블 7개" 일관: ✅
- §2.2 "FK 절대 추가 안 함" ↔ §4.2 "단일 진입점 트랜잭션" — 경합 없음
- CLAUDE.md 의 4규칙 ↔ ARCHITECTURE §4.14 ↔ 본 spec — 동일 메시지

### 7.3 Scope check

- 본 spec 의 범위는 **정책 결정과 그 즉각적 결과** 까지. retention 정책·CI drift 검출은 Out of scope (별도 ROADMAP 항목)

### 7.4 Ambiguity

| 잠재 ambiguity | 해소 방법 |
|---|---|
| "사람이 적용한다" — 운영자가 누구인가? | 현 단계: 솔로 개발자. 향후 팀 확장 시 "DB owner" 역할 정의 필요 (보류) |
| "FK 절대 금지" — `UNIQUE` / `PRIMARY KEY` 도? | NO. 본 정책의 "FK" 는 RDBMS `FOREIGN KEY` 절만. PK / UNIQUE / CHECK / NOT NULL 은 정상 사용 |
| "자동 migration 도구" — DB seed 데이터 스크립트도? | NO. 정책은 schema (DDL) 만 다룸. seed (DML) 는 별도 |

### 7.5 발견된 결함 (이 검토에서 새로 잡힌 것)

**D-1 (Medium):** `game_results` 데이터 retention 도입 시 자식 (`game_result_players`) 삭제 순서를 코드가 명시해야 함. FK CASCADE 가 자동 처리하던 부분을 누가 책임지는지 spec 화 필요. → ROADMAP T2-0c 로 등록 권장.

**D-2 (Low):** Backend 기동 시 sentinel `SELECT 1 FROM rooms LIMIT 1` 같은 스키마 검증 부재. 사람이 DDL 적용을 잊으면 첫 query 까지 에러가 지연됨. README 가이드로 1차 방어, 사고 시 도입 — 현 단계 보류.

**D-3 (Low):** ORM·migration 도구를 미래에 도입할 때 FK 자동 추가 옵션을 끄는 코드 리뷰 가이드 부재. 도입 시점 이슈로 deferred.

이 중 **D-1 은 즉시 ROADMAP 반영**한다. D-2 / D-3 는 spec 본문에 기록만.

---

## 8. Migration from Old State (이번 변경의 즉각 적용)

기존 운영 DB 가 없으므로 마이그레이션 부담 없음.

기존 dev 환경에서 이미 backend/migrations 를 통해 적용된 DB 가 있다면:
- 데이터: 그대로 유지 (DDL 호환)
- `schema_migrations` 메타 테이블: 잔존 가능. 무해. `DROP TABLE schema_migrations;` 로 정리 가능 (선택)

---

## 9. 검토 이력

| 날짜 | 이벤트 | 결과 |
|-----|-------|------|
| 2026-04-25 | retroactive 작성 + 자체 검토 1회 | 결함 D-1 (medium) / D-2 / D-3 (low) 발견. D-1 은 ROADMAP 반영 commit follow-up |
