## Why

서버가 동작하지만 플레이어가 게임에 참여할 UI가 없다. React SPA로 프론트엔드를 구축하여 실제로 플레이 가능한 서비스를 완성한다. `add-initial-state-api` 완료 이후 적용된다.

## What Changes

- `frontend/` 디렉토리 신규 생성 (Vite + React 18 + TypeScript)
- 로비(방 목록/생성/참가), 대기실, 게임(페이즈별 UI), 결과 화면 구현
- Zustand 기반 전역 상태에 WS 이벤트 반영
- playerID를 localStorage에 보관하여 재연결 시 세션 복원
- WS 자동 재연결 (grace period 30초 활용)

## Capabilities

### New Capabilities

- `lobby-ui`: 공개 방 목록 조회, 방 생성, 코드로 비공개 방 참가
- `waiting-room-ui`: 대기실 플레이어 목록, 방장의 게임 시작 버튼
- `game-ui`: 페이즈별 게임 화면 — 채팅, 투표, 밤 행동, 타이머, 역할 표시
- `result-ui`: 게임 종료 결과 화면 — 승리 팀, 플레이어별 역할 공개

### Modified Capabilities

없음

## Impact

- 신규 디렉토리: `frontend/` (백엔드 코드 변경 없음)
- 외부 의존성: Vite, React 18, TypeScript, TailwindCSS, Zustand
- 서버 API 소비: `GET/POST /api/rooms`, `POST /api/rooms/:id/join`, `POST /api/rooms/join/code`, `POST /api/rooms/:id/start`
- WS 소비: `ws://localhost:3000/ws/rooms/:id?player_id=xxx`
