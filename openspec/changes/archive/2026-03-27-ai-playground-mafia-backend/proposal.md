## Why

AI와 함께 다양한 게임을 즐길 수 있는 플랫폼이 없다. 마피아처럼 사람 간 심리전이 핵심인 게임을 혼자서도 즐길 수 있도록, LLM 기반 AI 플레이어가 실제 역할을 수행하는 플랫폼을 구축한다.

## What Changes

- Go 1.23 + Fiber 기반 백엔드 서버 신규 구축
- GameModule 인터페이스로 게임을 플러그인처럼 등록하는 플랫폼 구조 도입
- 마피아 게임 첫 번째 게임으로 구현 (항상 6명 고정, 사람 1~6명 + 나머지 AI)
- 각 AI 플레이어가 독립 goroutine + 독립 Claude API 세션으로 동작하는 AI Agent 시스템 구축
- 역할별 정보 격리: 마피아 AI는 공범만 알고, 경찰 AI는 조사 결과만 앎
- AI가 LLM 스스로 응답 여부를 판단하여 사람처럼 자연스럽게 참여
- 랜덤 한국 이름 + 성격 페르소나로 AI가 사람처럼 보이게 구성
- 낮(토론→투표) / 밤(역할 행동) 페이즈 관리, 페이즈별 타이머 자동 전환
- 공개/비공개 방 선택 가능, 비공개 방은 6자리 코드로 참가
- 플레이어 이탈 시 AI 자동 대체, 사람 전원 이탈 시 게임 종료
- PostgreSQL (docker-compose) + Zap 로깅 + TOML 설정 관리

## Capabilities

### New Capabilities

- `platform-core`: GameModule 인터페이스, 방 생성/관리(공개·비공개), 방장 관리, WebSocket 허브
- `ai-agent`: 독립 goroutine 기반 AI 에이전트, Claude API 세션 관리, 정보 격리, 자율 응답 판단, 발언 타이밍 제어
- `mafia-game`: 마피아 게임 로직 — 역할 배분, 낮/밤 페이즈 타이머, 투표, 마피아 비공개 채널, 승리 조건
- `persona-pool`: 랜덤 한국 이름 + 성격 페르소나 풀 관리

### Modified Capabilities

(없음 — 신규 프로젝트)

## Impact

- 신규 프로젝트이므로 기존 코드에 대한 영향 없음
- 외부 의존성: Claude API (Anthropic), PostgreSQL
- 추후 추가될 게임은 `internal/games/` 아래 GameModule 구현체만 추가하면 됨
- 인증 기능은 이번 범위에서 제외 (추후 추가 예정)
- 프론트엔드는 이번 범위에서 제외 (추후 추가 예정)
