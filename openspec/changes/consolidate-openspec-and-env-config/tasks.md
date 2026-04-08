## 1. openspec 통합

- [x] 1.1 `backend/openspec/specs/game-lifecycle/spec.md`를 `openspec/specs/game-lifecycle/spec.md`로 이동
- [x] 1.2 `backend/openspec/specs/phase-manager/spec.md`를 `openspec/specs/phase-manager/spec.md`로 이동
- [x] 1.3 `openspec/specs/ai-agent/spec.md`에 backend 버전의 "AI agent action functions respect context cancellation" Requirement 및 3개 Scenario 추가
- [x] 1.4 `backend/openspec/changes/archive/2026-04-06-fix-game-bugs/`를 `openspec/changes/archive/2026-04-06-fix-game-bugs/`로 이동
- [x] 1.5 `backend/openspec/` 디렉토리 삭제

## 2. ANTHROPIC_API_KEY → config.toml

- [x] 2.1 `config/config.go`의 `AIConfig` 구조체에 `APIKey string \`toml:"api_key"\`` 필드 추가
- [x] 2.2 `config.toml`의 `[ai]` 섹션에 `api_key = "<값>"` 추가 (주석 처리된 기존 키 값 사용)
- [x] 2.3 `cmd/server/main.go`에서 `apiKey := os.Getenv("ANTHROPIC_API_KEY")` 블록 제거
- [x] 2.4 `cmd/server/main.go`에서 `aiManager` 생성 시 `apiKey` 인자를 `cfg.AI.APIKey`로 교체
- [x] 2.5 `cmd/server/main.go`에서 `cfg.AI.APIKey == ""` 기동 시 검증 추가 (`logger.Fatal("ai.api_key is not set in config")`)

## 3. 빌드 검증

- [x] 3.1 `go build ./...` 컴파일 오류 없음 확인
- [ ] 3.2 서버 기동 후 AI 응답 정상 동작 확인
