## 1. 프로젝트 초기 설정

- [x] 1.1 `frontend/` 디렉토리에 `npm create vite@latest . -- --template react-ts` 실행
- [x] 1.2 의존성 추가: `zustand`, `react-router-dom`, `tailwindcss`, `postcss`, `autoprefixer`
- [x] 1.3 TailwindCSS 초기화 (`npx tailwindcss init -p`), `tailwind.config.js` + `index.css` 설정
- [x] 1.4 `src/api.ts` — 서버 API 호출 함수 작성 (`createRoom`, `listRooms`, `joinRoom`, `joinByCode`, `startGame`, `restartGame`)
- [x] 1.5 `src/types.ts` — 공유 타입 정의 (`Room`, `Player`, `GameSnapshot`, `WsEvent`, `Phase`, `GameOverResult` 등)

## 2. Zustand 스토어 + WS 훅

- [x] 2.1 `src/store/gameStore.ts` — Zustand 스토어 초기 구조 작성 (`playerID`, `myRole`, `room`, `phase`, `round`, `timerRemainingSec`, `alivePlayerIDs`, `votes`, `messages`, `wsStatus`, `result: GameOverResult | null`)
- [x] 2.2 `connect(roomID)` 액션 구현 — localStorage에서 playerID 읽기, WS URL 구성, WebSocket 생성
- [x] 2.3 `onopen` 핸들러 — `wsStatus: 'connected'` 설정
- [x] 2.4 `onmessage` 핸들러 — 이벤트 타입별 분기 처리
  - `initial_state`: 방·게임·역할 상태 전체 초기화, 타이머 시작
  - `role_assigned`: `myRole` 업데이트 (대기실에서 게임 시작 시 서버가 개별 전송)
  - `phase_change`: phase/round/timer/alivePlayerIDs 업데이트, 타이머 재시작
  - `chat` / `mafia_chat`: messages 배열에 추가
  - `vote`: votes 맵 업데이트
  - `kill`: alivePlayerIDs에서 제거, messages에 사망 알림 추가
  - `game_over`: `result` 필드에 `{ winner, round, duration_sec, players[] }` 저장 (서버 payload 구조 그대로 사용)
  - `night_action` (type: `investigation_result`): 경찰 플레이어에게 조사 결과 메시지 추가 (`is_mafia` 여부 표시)
  - `player_replaced`: messages에 알림 추가
- [x] 2.5 `onclose` 핸들러 — exponential backoff 자동 재연결 (1s → 2s → 4s → 최대 10s)
- [x] 2.6 `sendAction(type, payload)` 액션 — WS로 JSON 전송
- [x] 2.7 `disconnect()` 액션 — WS 닫기, 타이머 정리
- [x] 2.8 타이머 로직 — `setInterval` 기반 로컬 카운트다운, phase 변경 시 재시작

## 3. 라우터 + 앱 구조

- [x] 3.1 `src/main.tsx`에 `BrowserRouter` 설정
- [x] 3.2 `src/App.tsx` — `Routes` 정의: `/` → `LobbyPage`, `/rooms/:id` → `RoomPage`
- [x] 3.3 `src/pages/RoomPage.tsx` — `room.status`에 따라 `WaitingRoom` 또는 `GameRoom` 컴포넌트 렌더링, WS connect/disconnect 관리

## 4. 로비 화면

- [x] 4.1 `src/pages/LobbyPage.tsx` — 마운트 시 `listRooms()` 호출, 방 목록 렌더링
- [x] 4.2 방 목록 항목 컴포넌트 — 방 이름, 플레이어 수, 상태 표시, 클릭 시 닉네임 입력 모달 표시 후 `joinRoom()` 호출, `player_id` localStorage 저장 후 `/rooms/:id` 이동
- [x] 4.3 방 만들기 폼 — 방 이름 입력, 공개/비공개 선택, 닉네임 입력, `createRoom()` 호출 후 `/rooms/:id` 이동
- [x] 4.4 코드 참가 폼 — 코드 + 닉네임 입력, `joinByCode()` 호출 후 이동, 오류 메시지 표시

## 5. 대기실 화면

- [x] 5.1 `src/components/WaitingRoom.tsx` — 플레이어 목록 (이름, 방장 표시, AI 표시)
- [x] 5.2 비공개 방 코드 표시 (visibility === 'private')
- [x] 5.3 방장이면 "게임 시작" 버튼 표시, 클릭 시 `startGame()` 호출 (`X-Player-ID` 헤더 포함)
- [x] 5.4 일반 플레이어는 "방장이 게임을 시작하기를 기다리는 중..." 메시지 표시

## 6. 게임 화면

- [x] 6.1 `src/components/GameRoom.tsx` — 게임 전체 레이아웃 (페이즈 헤더, 플레이어 목록, 채팅, 액션 패널)
- [x] 6.2 `PhaseHeader` 컴포넌트 — 현재 페이즈 이름(한국어), 라운드, 타이머 카운트다운 표시
- [x] 6.3 `PlayerList` 컴포넌트 — 생존/사망 구분 표시, 사망 시 공개된 역할 표시
- [x] 6.4 `ChatLog` 컴포넌트 — 메시지 목록, 자동 스크롤
- [x] 6.5 `ChatInput` 컴포넌트 — 텍스트 입력 + 전송, day_discussion/day_vote 페이즈에서 활성화
- [x] 6.6 `VotePanel` 컴포넌트 — day_vote 페이즈에서 생존 플레이어 클릭으로 투표, 현재 투표 현황 표시
- [x] 6.7 `NightPanel` 컴포넌트 — night 페이즈 역할별 분기:
  - `mafia`: 생존 시민 목록에서 킬 대상 클릭 → `{ type: "kill", night: { action_type: "kill", target_id } }` 전송
  - `police`: 생존 플레이어 목록에서 조사 대상 클릭 → `{ type: "investigate", night: { action_type: "investigate", target_id } }` 전송; `night_action` 이벤트 수신 시 "X는 마피아입니다/아닙니다" 메시지 표시
  - 그 외: "밤입니다. 마피아가 활동 중..." 메시지만 표시
- [x] 6.8 마피아 채팅 입력 — night 페이즈 + mafia 역할일 때만 표시, `mafia_only: true`로 전송

## 7. 결과 화면

- [x] 7.1 `src/components/ResultOverlay.tsx` — game_over 이벤트 수신 시 오버레이로 표시
- [x] 7.2 승리 팀 강조 표시 (마피아 승 / 시민 승)
- [x] 7.3 플레이어별 이름 + 역할 + 생존 여부 목록 — `result.players[]` 배열 사용 (game_over payload에서 서버가 내려주는 값)
- [x] 7.4 방장이면 "다시 시작" 버튼 표시, `restartGame()` 호출 후 오버레이 닫기, `result` 스토어 초기화
- [x] 7.5 "나가기" 버튼 표시 — localStorage `player_id_${roomID}` 삭제 후 `/`로 이동

## 8. 마무리

- [x] 8.1 `frontend/vite.config.ts`에 개발 서버 프록시 설정 (`/api`, `/ws` → `http://localhost:3000`)
- [x] 8.2 전체 흐름 수동 확인: 방 생성 → 참가 → 게임 시작 → 채팅 → 투표 → 결과
