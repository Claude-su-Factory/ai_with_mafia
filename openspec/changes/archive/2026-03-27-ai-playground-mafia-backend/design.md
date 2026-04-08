## Context

신규 프로젝트. AI 플레이어와 실시간으로 마피아 게임을 즐길 수 있는 백엔드 서버를 구축한다. 핵심 제약은 두 가지다: (1) 여러 AI 플레이어가 각자의 역할 정보만 알아야 하는 정보 격리, (2) 추후 다른 게임을 쉽게 추가할 수 있는 확장 가능한 플랫폼 구조.

## Goals / Non-Goals

**Goals:**
- GameModule 인터페이스로 게임을 플러그인처럼 등록하는 플랫폼 구조
- 각 AI 플레이어가 독립 goroutine + 독립 Claude API 세션으로 동작
- 마피아 게임 완전 구현 (역할 배분, 낮/밤 페이즈 타이머, 투표, 승리 조건)
- WebSocket 기반 실시간 통신
- TOML 설정으로 Claude 모델 전환 가능
- 공개/비공개 방, 방장 관리, 플레이어 이탈 처리

**Non-Goals:**
- 사용자 인증 (추후)
- 프론트엔드 (추후)
- 게임 히스토리 분석 기능
- AI 난이도 조절

## Decisions

### 1. GameModule 인터페이스 패턴

새 게임 추가 시 `internal/games/<game-name>/` 아래에 아래 인터페이스를 구현하면 플랫폼이 자동 인식한다.

```go
type GameModule interface {
    Name() string
    Config() GameConfig
    NewGame(room *entity.Room) Game
}

type Game interface {
    Start(ctx context.Context)
    HandleAction(playerID string, action Action) error
    State() GameState
    Subscribe() <-chan GameEvent
}
```

**대안 고려**: 설정 파일(YAML)만으로 게임 정의 → 게임마다 로직이 완전히 달라 코드 없이 표현 불가. 인터페이스 방식 채택.

---

### 2. AI Agent = 독립 goroutine + 독립 Claude API 세션

각 AI 플레이어는 별도 goroutine에서 실행되고, 자신만의 `[]Message` 히스토리를 유지한다. 게임 서버는 각 AI에게 역할에 맞는 컨텍스트만 전달한다.

```
Game Server (전체 진실 보유)
    │
    ├── AI Agent "김민준" goroutine
    │   system_prompt: "너는 마피아, 공범: 이지은"
    │   history: [공개 채팅 전체]
    │   private: {allies: ["이지은"]}
    │
    └── AI Agent "이지은" goroutine
        system_prompt: "너는 시민"
        history: [공개 채팅 전체] (동일 내용이지만 별도 세션)
```

**대안 고려**: 단일 LLM 호출로 모든 AI 응답 생성 → 역할 간 정보 누출 위험, 페르소나 일관성 유지 불가. 독립 세션 방식 채택.

---

### 3. AI 응답 여부 자율 판단

AI가 매 메시지에 응답할지 여부를 LLM 스스로 결정한다. 응답 또는 `[PASS]` 토큰을 반환하는 단일 API 호출로 처리한다. 별도 "응답할지 판단" 호출을 추가하지 않아 API 호출 수를 절약한다.

```
채팅 이벤트 수신
    │
    ▼
Claude API call: "이 상황에서 발언할 것인가? 발언한다면 내용은?"
    │
    ├── [PASS] → 응답 없음
    └── 실제 메시지 → 랜덤 딜레이 후 브로드캐스트
```

---

### 4. 동시 Claude API 호출 제어 (Semaphore)

AI 5명이 동시에 응답하면 rate limit 초과 가능. semaphore로 동시 호출 수를 제한한다.

```go
type AIManager struct {
    semaphore chan struct{} // config의 ai.max_concurrent 값
}
```

---

### 5. 모델 설정을 TOML로 분리

```toml
[ai]
model_default = "claude-haiku-4-5-20251001"   # 일반 채팅 응답
model_reasoning = "claude-sonnet-4-6"          # 투표 등 중요 판단
max_concurrent = 3
history_max = 40
response_delay_min = 2
response_delay_max = 6

[game.mafia.timers]
day_discussion = 300   # 낮 토론 5분
day_vote = 120         # 투표 2분
night = 60             # 밤 행동 1분
```

---

### 6. 게임 상태 보호: sync.RWMutex

여러 AI goroutine이 동시에 게임 상태를 읽고 쓰므로 `sync.RWMutex`로 보호한다. 읽기 다수 / 쓰기 소수 패턴이므로 RWMutex가 적합.

---

### 7. 데이터 저장: PostgreSQL + golang-migrate

방 정보, 게임 결과를 영속 저장한다. 로컬 개발은 docker-compose로 PostgreSQL 구동. DB 드라이버는 `pgx` 사용. 마이그레이션은 `golang-migrate`로 관리한다.

---

### 8. 방 공개/비공개

방 생성 시 공개(로비 목록 노출) 또는 비공개(6자리 랜덤 코드로만 참가) 선택. 비공개 코드는 대소문자 구분 없는 6자리 영숫자로 자동 생성.

---

### 9. 방장 관리 및 자동 이전

방장이 이탈하면 입장 순서 기준 다음 사람에게 방장 권한이 자동 이전된다. 모든 사람 플레이어가 이탈하면 방이 삭제된다. 게임 중 플레이어 이탈 시 해당 자리는 AI로 자동 대체되며 전체에게 알림이 전송된다.

---

### 10. 항상 6명 고정 + 재시작

마피아 게임은 항상 정확히 6명으로 진행된다. `max_humans`는 1~6 설정 가능하며, 부족한 자리는 AI로 채운다. `max_humans=6`이면 AI 없이 사람만으로 진행된다. 재시작 시 페르소나와 역할을 새로 랜덤 배정한다.

---

### 11. 프로젝트 구조

```
ai-playground/
├── cmd/server/main.go
├── config/
│   └── config.go              ← TOML 파싱 (BurntSushi/toml)
├── config.toml
├── docker-compose.yml
├── migrations/                ← golang-migrate SQL 파일
├── internal/
│   ├── domain/
│   │   ├── entity/            ← Player, Room, Game (도메인 객체, mutex 포함)
│   │   └── dto/               ← API/WS 요청·응답 (직렬화 가능)
│   ├── platform/
│   │   ├── registry.go        ← GameModule 등록
│   │   ├── room.go            ← 방 생성/관리
│   │   └── ws/hub.go          ← WebSocket 허브
│   ├── ai/
│   │   ├── agent.go           ← AI 에이전트 goroutine
│   │   ├── manager.go         ← 동시 호출 제어, 발언 타이밍
│   │   └── persona.go         ← 랜덤 이름/성격 풀
│   └── games/
│       └── mafia/
│           ├── game.go
│           ├── roles.go
│           └── phases.go
└── go.mod
```

## Risks / Trade-offs

- **Goroutine 누수** → 모든 AI Agent goroutine은 `context.Context`를 인자로 받고, 방 종료 시 `cancel()` 호출로 일괄 정리
- **AI 히스토리 토큰 초과** → `history_max` 설정으로 오래된 메시지 제거 (system prompt는 항상 유지)
- **AI 응답 지연** → Claude API latency로 실시간감 저하 가능. `response_delay` 설정으로 자연스러운 딜레이 연출
- **Entity 직렬화** → mutex/channel 포함 Entity를 직접 JSON 직렬화하면 panic. DTO 변환 레이어 필수
- **PostgreSQL 없이 서버 시작** → docker-compose 미실행 시 서버 시작 실패. 로컬 개발 가이드 필요
- **마피아 밤 투표 무효** → 마피아끼리 처치 대상 의견이 갈리면 그날 밤 아무도 안 죽음. 게임 밸런스 상 의도된 동작
