## Why

두 가지 구조적 문제가 공존한다.

첫째, `backend/openspec/`이 root의 `openspec/`과 별도로 존재한다. `backend/` 디렉토리 안에서 opsx CLI를 실행하면서 자동 생성된 것으로, `game-lifecycle`, `phase-manager`, `ai-agent` spec과 `fix-game-bugs` 아카이브가 root에 반영되지 않은 채 분리되어 있다. 이 상태를 그대로 두면 앞으로 backend 관련 opsx 작업이 어느 openspec에 기록되어야 하는지 불분명해진다.

둘째, `ANTHROPIC_API_KEY`를 `os.Getenv`로 읽는다. 다른 모든 설정값은 `config.toml`에서 관리하는데 API key만 환경 변수에서 읽는 것은 일관성이 없다. config.toml 주석에도 이미 API key가 남아 있어 원래 toml에서 관리하려 했던 의도가 보인다.

## What Changes

- `backend/openspec/` 삭제, 내용을 root `openspec/`으로 통합
  - `game-lifecycle`, `phase-manager` spec → root로 이동
  - `ai-agent` spec → root 버전에 backend의 context cancellation 요구사항 병합
  - `fix-game-bugs` 아카이브 → root `openspec/changes/archive/`로 이동
- `config.toml`의 `[ai]` 섹션에 `api_key` 필드 추가
- `config/config.go`의 `AIConfig` 구조체에 `APIKey string` 필드 추가
- `cmd/server/main.go`에서 `os.Getenv("ANTHROPIC_API_KEY")` 제거, `cfg.AI.APIKey` 사용

## Capabilities

### New Capabilities

없음.

### Modified Capabilities

- `ai-agent`: context cancellation 요구사항 추가 (backend 버전에서 병합)

## Impact

- `config/config.go`: `AIConfig`에 `APIKey` 필드 추가
- `config.toml`: `[ai]` 섹션에 `api_key` 추가, 기존 주석 처리된 키 값 활용
- `cmd/server/main.go`: `os.Getenv("ANTHROPIC_API_KEY")` 제거, 기동 시 `cfg.AI.APIKey == ""` 검증으로 대체
- `openspec/specs/ai-agent/spec.md`: context cancellation requirement 병합
- `openspec/specs/game-lifecycle/`, `openspec/specs/phase-manager/`: 신규 추가
- `backend/openspec/`: 삭제

## Non-goals

- `CONFIG_PATH` 환경 변수는 제거하지 않는다. config 파일 경로 자체를 부트스트랩하는 메커니즘으로, config.toml에 담을 수 없는 성질의 값이다.
- `config.toml`의 `database.dsn`이나 `redis.password` 등 다른 민감 값의 관리 방식은 이번 작업 범위에 포함하지 않는다.
