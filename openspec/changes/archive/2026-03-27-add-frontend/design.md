## Context

백엔드는 REST API + WebSocket으로 구성되어 있다. WS 연결 시 `initial_state` 이벤트로 전체 상태를 한 번에 내려주며, 이후 게임 이벤트(`phase_change`, `chat`, `vote`, `kill`, `game_over`)가 실시간으로 전달된다. playerID는 서버가 join 응답에서 UUID로 발급하며, 클라이언트가 보관해야 한다.

## Goals / Non-Goals

**Goals:**
- 마피아 게임을 처음부터 끝까지 플레이할 수 있는 UI
- 페이지 새로고침 후 게임 상태 복원 (initial_state 활용)
- WS 자동 재연결 (grace period 30초)

**Non-Goals:**
- 모바일 최적화 (데스크톱 우선)
- Google Auth / 계정 시스템 (추후 별도 change)
- 게임 결과 히스토리 조회 페이지
- 다크모드

## Decisions

### 1. 라우팅: React Router v6, URL에 roomID 포함

```
/                    → 로비 (방 목록)
/rooms/:id           → 대기실 또는 게임 (방 상태에 따라 자동 전환)
```

대기실과 게임을 하나의 라우트(`/rooms/:id`)로 통합. `room.status`에 따라 컴포넌트를 분기한다. 결과 화면도 `game_over` 이벤트 수신 시 같은 라우트 내에서 오버레이로 표시.

### 2. 상태 관리: Zustand 단일 스토어

```
useGameStore {
  // 세션
  playerID: string               // localStorage 동기화
  myRole: string                 // initial_state 또는 role_assigned 이벤트에서 수신

  // 방
  room: RoomState | null         // id, name, status, hostID, visibility, joinCode, players[]

  // 게임 (playing 중에만)
  phase: Phase | null
  round: number
  timerRemainingSec: number
  alivePlayerIDs: string[]
  votes: Record<string, string>

  // 결과 (game_over 이후)
  result: GameOverResult | null  // { winner, round, duration_sec, players[] }

  // 채팅
  messages: ChatMessage[]

  // WS
  wsStatus: 'connecting' | 'connected' | 'reconnecting' | 'disconnected'

  // actions
  connect(roomID): void
  disconnect(): void
  sendAction(type, payload): void
}
```

WS 연결 로직과 이벤트 핸들러를 스토어 내부에 캡슐화. 컴포넌트는 상태만 구독.

### 3. WS 자동 재연결

```
연결 끊김 감지
    │
    ├─ 1초 후 재연결 시도
    ├─ 실패 시 exponential backoff (최대 10초)
    └─ 30초(grace period) 이내 재연결 시 서버가 세션 복원
```

재연결 성공 시 `initial_state` 이벤트를 다시 수신하여 스토어 상태를 재동기화. 별도 HTTP 상태 조회 불필요.

### 4. 타이머: 클라이언트 로컬 카운트다운

`initial_state` 또는 `phase_change` 수신 시 `timer_remaining_sec`으로 로컬 `setInterval` 시작. 서버 동기화는 reconnect 시 initial_state로만 한다. 1초 단위 카운트다운으로 충분.

### 5. 역할 민감 정보 처리

`my_role`은 `initial_state`에서만 수신. 마피아 채팅(`mafia_only: true`)은 서버가 마피아 클라이언트에게만 WS로 전송하므로 프론트는 수신된 메시지를 그대로 표시하면 됨.

### 6. playerID 저장: localStorage

```
key: `player_id_${roomID}`
value: UUID (서버 발급)
```

방마다 별도 key를 사용해 여러 탭에서 다른 방 참가 가능.

localStorage 삭제 시점:
- `game_over` 이벤트 수신 후 결과 화면에서 "다시 시작" 또는 "나가기"를 누를 때
- 로비(`/`)로 명시적으로 이동할 때

페이지 새로고침이나 탭 닫기에서는 유지 → grace period 재연결에 활용.

## Risks / Trade-offs

- **localStorage 보안**: XSS 취약점 존재. Auth 없는 지금은 허용 범위, Google Auth 붙으면 JWT 쿠키로 교체.
- **타이머 drift**: 로컬 카운트다운은 정확하지 않다. reconnect 시 서버 값으로 재동기화하므로 오차가 누적되지 않음.
- **Zustand vs Redux**: Zustand가 WS 이벤트 핸들러를 스토어에 넣기 더 편하고 보일러플레이트가 적음. 이 규모에서는 Zustand로 충분.
