---
name: frontend-dev
description: "AI 마피아 게임 React/TypeScript 프론트엔드 개발 스킬. UI 컴포넌트, Zustand 상태 관리, WS 이벤트 핸들링, API 클라이언트 작업을 수행한다. React, TypeScript, Vite, Zustand 관련 코드 작업 시 반드시 이 스킬을 사용할 것. 다시 실행, 업데이트, UI 수정 요청 시에도 이 스킬을 사용."
---

# Frontend Dev Skill

AI 마피아 게임 React/TypeScript 프론트엔드 작업을 위한 가이드.

## 작업 흐름

1. 요청 분석 → 영향받는 파일 파악 (store / components / pages / api.ts / types.ts)
2. 관련 파일 Read → 현재 구현 이해
3. 변경 구현 → Edit/Write 사용
4. 결과 보고 (타입 체크 이슈 포함)

## 핵심 패턴

### WS 액션 전송

`sendAction(type, payload)` 함수는 `{ type, ...payload }` 형태로 직렬화된다.
백엔드 `dto.ActionRequest`는 중첩 구조이므로 payload에 반드시 올바른 키 사용:

```ts
// 채팅
sendAction('chat', { chat: { message: trimmed } })

// 투표  
sendAction('vote', { vote: { target_id: playerID } })

// 밤 행동
sendAction('kill', { night: { action_type: 'kill', target_id: playerID } })
sendAction('investigate', { night: { action_type: 'investigate', target_id: playerID } })
```

### 새 WS 이벤트 처리

`gameStore.ts`의 `ws.onmessage` switch문에 케이스 추가:

```ts
case 'new_event_type': {
    const { field1, field2 } = event.payload
    set((s) => ({ /* state update */ }))
    break
}
```

### types.ts 타입 정의

백엔드 JSON은 snake_case. TypeScript 타입도 snake_case를 그대로 사용한다 (camelCase 변환 금지):

```ts
// ✅ 올바른 패턴
interface Room {
    id: string
    host_id: string     // snake_case 유지
    max_humans: number  // snake_case 유지
    players: Player[]
    status: 'waiting' | 'playing' | 'finished'
}

// ❌ 잘못된 패턴
interface Room {
    hostId: string      // camelCase — 백엔드와 불일치
    maxHumans: number   // camelCase — 백엔드와 불일치
}
```

### localStorage player_id 패턴

방 생성/입장 시:
```ts
localStorage.setItem(`player_id_${res.id}`, res.player_id)
navigate(`/rooms/${res.id}`)
```

방 페이지 진입 시:
```ts
const playerID = localStorage.getItem(`player_id_${roomID}`) ?? ''
if (!playerID) navigate('/lobby')
```

### Vite proxy 설정

`vite.config.ts` 기준:
```ts
'/ws': { target: 'ws://localhost:3000', ws: true, changeOrigin: true }
'/api': 'http://localhost:3000'
```

## 상태 관리 (gameStore.ts)

| 상태 | 타입 | 업데이트 이벤트 |
|------|------|----------------|
| `room` | `Room \| null` | `initial_state`, `phase_change` |
| `myRole` | `Role` | `initial_state`, `role_assigned` |
| `phase` | `Phase \| null` | `phase_change` |
| `alivePlayerIDs` | `string[]` | `phase_change`, `kill` |
| `votes` | `Record<string, string>` | `vote`, `phase_change`(리셋) |
| `messages` | `ChatMessage[]` | `chat`, `mafia_chat`, `kill`, `night_action` |
| `result` | `GameOverResult \| null` | `game_over` |
| `wsStatus` | `'connecting'\|'connected'\|'reconnecting'\|'disconnected'` | WS 이벤트 |

## 컴포넌트별 주요 로직

### WaitingRoom.tsx
- `canStart`: `room.players.length >= 1` (AI 제외된 players 기준)
- 방장만 "게임 시작" 버튼 표시: `room.host_id === playerID`

### LobbyPage.tsx
- 방 생성: `createRoom({ name, visibility, player_name })` → `player_id`, `id` 반환
- 방 목록: `room.players.length`는 AI 제외된 인원수 (백엔드가 필터링)

### GameRoom.tsx
- 현재 페이즈에 따라 `VotePanel` 또는 `NightPanel` 표시
