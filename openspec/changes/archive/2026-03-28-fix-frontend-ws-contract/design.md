## Context

fix-server-ws-contract 이후 WS 이벤트 구조가 바뀌었고, 동시에 초기 구현 시 서버 API 계약을 잘못 파악한 채로 작성된 코드 두 군데가 있다.

현재 문제:
1. `chat`/`mafia_chat` 핸들러: `event.player_id`, `event.message` flat 접근 → 서버는 이제 `event.payload.sender_id`, `event.payload.sender_name`, `event.payload.message` 사용
2. `phase_change` 핸들러: `room.status`를 업데이트하지 않아 게임 시작 후 WaitingRoom이 계속 렌더됨
3. `createRoom`: `game_type`, `max_humans` 누락; `player_name`을 body에 넣지만 서버는 `X-Player-Name` 헤더로 읽음; 응답 `res.room_id` 대신 `res.id` 사용해야 함
4. `listRooms`: 서버가 배열 직접 반환, 프론트는 `{rooms: []}` 객체로 파싱 시도

## Goals / Non-Goals

**Goals:**
- 서버 변경 이후 chat/mafia_chat 이벤트 올바르게 수신
- phase_change 수신 시 GameRoom으로 화면 전환
- createRoom, listRooms API 호출이 정상 동작
- ChatLog에서 `sender_name` 활용해 room.players 조회 불필요

**Non-Goals:**
- 서버 코드 변경
- 새 기능 추가
- UI 디자인 변경

## Decisions

### chat/mafia_chat: WsEvent 타입과 핸들러 동시 수정

**결정**: `types.ts`의 WsEvent union 타입과 `gameStore.ts` 핸들러를 함께 수정한다. 타입을 먼저 바꾸면 TypeScript 컴파일러가 핸들러의 잘못된 접근을 오류로 잡아준다.

```ts
// 변경 전
| { type: 'chat'; player_id: string; message: string; mafia_only?: boolean }

// 변경 후
| { type: 'chat'; payload: { sender_id: string; sender_name: string; message: string; mafia_only?: boolean } }
```

`ChatMessage`에는 이미 `player_name?: string` 필드가 선언되어 있으므로 `sender_name`을 바로 저장할 수 있다. `ChatLog`는 `player_name`이 있으면 그걸 쓰고, 없으면 기존처럼 `room.players` 조회로 폴백한다.

### phase_change: room 객체 직접 업데이트

**결정**: `phase_change` 핸들러에서 `room` 객체의 `status`를 `'playing'`으로 업데이트한다.

```ts
case 'phase_change': {
  // 기존 코드 ...
  set((s) => ({
    ...updates,
    room: s.room ? { ...s.room, status: 'playing' } : s.room,
  }))
}
```

**대안**: `initial_state`에서 이미 `room.status === 'playing'`으로 오는 경우를 믿는다.
**왜 기각**: 게임 시작 시 연결된 플레이어는 `initial_state` 이후 `phase_change`를 받는데, `initial_state`의 room.status가 이미 `'playing'`이라면 문제없다. 하지만 **WaitingRoom에서 게임이 시작될 때** 프론트는 이미 연결된 상태이므로 `initial_state`를 다시 받지 않고 `phase_change`만 받는다. 따라서 phase_change 핸들러에서 room.status를 업데이트하는 것이 필수다.

### createRoom: game_type/max_humans 하드코딩

**결정**: 현재 게임 타입이 `mafia` 하나뿐이고 인원 설정 UI가 없으므로 `game_type: 'mafia'`, `max_humans: 1`을 api.ts에서 하드코딩한다.

**대안**: UI에서 입력받는다.
**왜 기각**: 현재 요구사항에 없다. 나중에 추가할 수 있도록 `CreateRoomParams`에 optional 필드로 남긴다.

`player_name`은 `CreateRoomParams`에 유지하되 body JSON에서 제거하고 `X-Player-Name` 헤더로 전달한다. LobbyPage의 `createRoom` 호출부는 변경 불필요. 응답의 room ID는 `res.room_id` → `res.id`로 수정한다.

### listRooms: 반환 타입을 배열로 변경

**결정**: `request<Room[]>('/rooms')`로 변경하고 `LobbyPage`에서 `data.rooms` 대신 `data`를 직접 사용한다.

## Risks / Trade-offs

- **`request` 헬퍼 헤더 병합 버그**: `{ headers: {...}, ...options }` 패턴에서 `options.headers`가 전체 `headers` 키를 덮어쓴다. body와 커스텀 헤더를 동시에 쓰는 `createRoom`에서 `Content-Type`이 사라져 서버 파싱 실패로 이어진다. `request` 함수 내부에서 headers를 `restOptions`와 분리해서 병합해야 한다.
- **createRoom 하드코딩**: `game_type: 'mafia'`와 `max_humans: 1`을 하드코딩하므로 게임 타입이 늘어나면 수정이 필요하다. 현재 단계에서는 허용.
- **ChatLog 폴백 로직 유지**: `player_name`이 없는 메시지(시스템 메시지 등)는 기존처럼 `getPlayerName` 함수를 사용하므로 하위 호환성 유지.
