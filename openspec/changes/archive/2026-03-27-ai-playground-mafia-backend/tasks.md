## 1. 프로젝트 초기 설정

- [x] 1.1 Go 1.23 모듈 초기화 (`go mod init ai-playground`)
- [x] 1.2 의존성 추가: fiber, zap, BurntSushi/toml, pgx, golang-migrate, anthropic-go SDK
- [x] 1.3 `config.toml` 작성 (서버 포트, DB DSN, AI 모델/파라미터, 페르소나 풀, 게임 타이머)
- [x] 1.4 `config/config.go` TOML 파싱 구현
- [x] 1.5 `docker-compose.yml` PostgreSQL 서비스 정의
- [x] 1.6 `cmd/server/main.go` Fiber 앱 기본 구조 + Zap 로거 초기화

## 2. 도메인 Entity / DTO 정의

- [x] 2.1 `internal/domain/entity/player.go` — Player 엔티티 (ID, 이름, 역할, 생존 여부, AI 여부)
- [x] 2.2 `internal/domain/entity/room.go` — Room 엔티티 (ID, 이름, 게임 타입, 공개여부, 참가코드, 방장 ID, 플레이어 목록, 상태, mutex)
- [x] 2.3 `internal/domain/entity/game.go` — Game 인터페이스 및 GameState, GameEvent, Phase 타입
- [x] 2.4 `internal/domain/dto/room.go` — CreateRoomRequest(이름/타입/max_humans/공개여부), RoomResponse, JoinRoomRequest(코드/닉네임) DTO
- [x] 2.5 `internal/domain/dto/game.go` — GameEventDTO, ActionRequest, ChatMessageDTO, VoteRequest

## 3. Platform Core — GameModule 인터페이스 및 레지스트리

- [x] 3.1 `internal/platform/registry.go` — GameModule 인터페이스 정의 및 Register/Get 구현
- [x] 3.2 `internal/platform/room.go` — 방 생성(공개/비공개), 참가, 조회, 종료 로직 구현
- [x] 3.3 `internal/platform/room.go` — 비공개 방 6자리 코드 생성 로직
- [x] 3.4 `internal/platform/room.go` — 방장 이전, 빈 방 자동 삭제 로직
- [x] 3.5 방 생성 API (`POST /api/rooms`) Fiber 핸들러
- [x] 3.6 방 참가 API (`POST /api/rooms/:id/join`, `POST /api/rooms/join/code`) 핸들러
- [x] 3.7 공개 방 목록 API (`GET /api/rooms`) — 공개 방만 반환
- [x] 3.8 방 상세 조회 API (`GET /api/rooms/:id`)
- [x] 3.9 게임 시작 API (`POST /api/rooms/:id/start`) — 방장 권한 검증 포함
- [x] 3.10 게임 재시작 API (`POST /api/rooms/:id/restart`) — 방장 권한 검증 포함

## 4. WebSocket 허브

- [x] 4.1 `internal/platform/ws/hub.go` — WebSocket 허브 (Register/Unregister/Broadcast) 구현
- [x] 4.2 `internal/platform/ws/hub.go` — 방별 클라이언트 그룹 관리
- [x] 4.3 `internal/platform/ws/hub.go` — 마피아 비공개 채널 브로드캐스트 구현
- [x] 4.4 WebSocket 엔드포인트 (`GET /ws/rooms/:id`) Fiber 핸들러 연결
- [x] 4.5 클라이언트 이탈 시 플레이어 AI 대체 + 전체 알림 처리
- [x] 4.6 모든 사람 플레이어 이탈 시 게임 종료 처리

## 5. 페르소나 풀

- [x] 5.1 `internal/ai/persona.go` — 페르소나 구조체 (이름, 성격) 정의
- [x] 5.2 config.toml에서 페르소나 풀 로드, 없으면 기본 내장 풀 사용
- [x] 5.3 게임 시작/재시작 시 중복 없이 랜덤 페르소나 배정 로직 구현

## 6. AI Agent

- [x] 6.1 `internal/ai/agent.go` — Agent 구조체 정의 (페르소나, 역할, system prompt, 히스토리, eventCh)
- [x] 6.2 `internal/ai/agent.go` — `Run(ctx context.Context)` goroutine 루프 구현
- [x] 6.3 역할별 system prompt 생성 함수 구현 (마피아/시민/경찰 각각, AI임을 밝히지 않도록 명시)
- [x] 6.4 히스토리 최대 크기 제한 (history_max) 로직 구현
- [x] 6.5 `internal/ai/manager.go` — semaphore 기반 동시 API 호출 제어 구현
- [x] 6.6 `internal/ai/manager.go` — 랜덤 딜레이 + 순차 발언 타이밍 제어 구현
- [x] 6.7 채팅 이벤트 시 LLM이 응답 여부 자율 판단 구현 (응답 또는 [PASS] 반환, model_default 사용)
- [x] 6.8 투표/밤 행동 판단 호출 구현 (model_reasoning 사용)

## 7. 마피아 게임 구현

- [x] 7.\1 `internal/games/mafia/roles.go` — 역할 타입 및 고정 배분 비율 정의 (마피아2/경찰1/시민3)
- [x] 7.\1 `internal/games/mafia/roles.go` — 게임 시작 시 역할 무작위 배분 로직
- [x] 7.\1 `internal/games/mafia/phases.go` — Phase 타입 (낮토론/투표/밤/결과) 상태 머신
- [x] 7.\1 `internal/games/mafia/phases.go` — 낮 토론 페이즈: 공개 채팅 브로드캐스트 + 300초 타이머 → 자동 투표 전환
- [x] 7.\1 `internal/games/mafia/phases.go` — 투표 페이즈: 투표 현황 실시간 공개, 120초 타이머, 동표 무효 처리
- [x] 7.\1 `internal/games/mafia/phases.go` — 밤 페이즈: 마피아 비공개 채널 오픈, 60초 타이머
- [x] 7.\1 `internal/games/mafia/phases.go` — 마피아 처치 투표: 전원 일치 시 처치 확정, 불일치 시 무효
- [x] 7.\1 `internal/games/mafia/phases.go` — 경찰 조사: 조사 결과 경찰에게만 비공개 전달
- [x] 7.\1 `internal/games/mafia/game.go` — 승리 조건 판정 (시민 승리 / 마피아 승리)
- [x] 7.\1 `internal/games/mafia/game.go` — GameModule 인터페이스 구현 및 registry 등록
- [x] 7.\1 게임 종료 시 결과를 PostgreSQL에 저장

## 8. PostgreSQL 연동

- [x] 8.\1 DB 연결 초기화 및 ping 확인 (pgx 사용)
- [x] 8.\1 `migrations/` 디렉토리 생성 및 golang-migrate 설정
- [x] 8.\1 rooms 테이블 마이그레이션 파일 작성 (이름, 공개여부, 참가코드, 방장 ID 포함)
- [x] 8.\1 game_results 테이블 마이그레이션 파일 작성
- [x] 8.\1 Room CRUD 리포지토리 구현
- [x] 8.\1 GameResult 저장 리포지토리 구현

## 9. 통합 및 마무리

- [x] 9.1 `main.go`에서 모든 컴포넌트 조립 (config → DB 마이그레이션 → registry → Fiber 앱)
- [x] 9.2 `go run -race`로 race condition 없음 확인
- [x] 9.3 로컬 실행 가이드 작성 (docker-compose up → go run 순서)
