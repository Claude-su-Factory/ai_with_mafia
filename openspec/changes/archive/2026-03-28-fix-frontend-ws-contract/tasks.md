## 1. WsEvent 타입 수정

- [x] 1.1 `frontend/src/types.ts`의 `WsEvent` union에서 `chat` 타입을 `{ type: 'chat'; payload: { sender_id: string; sender_name: string; message: string; mafia_only?: boolean } }`로 변경
- [x] 1.2 `frontend/src/types.ts`의 `WsEvent` union에서 `mafia_chat` 타입을 `{ type: 'mafia_chat'; payload: { sender_id: string; sender_name: string; message: string } }`로 변경

## 2. gameStore chat/mafia_chat 핸들러 수정

- [x] 2.1 `gameStore.ts` `chat` 핸들러에서 `event.player_id` → `event.payload.sender_id`, `event.message` → `event.payload.message`, `event.mafia_only` → `event.payload.mafia_only`로 변경
- [x] 2.2 `gameStore.ts` `chat` 핸들러에서 `player_name: event.payload.sender_name` 저장 추가
- [x] 2.3 `gameStore.ts` `mafia_chat` 핸들러에서 동일하게 `event.payload.sender_id`, `event.payload.sender_name`, `event.payload.message`로 변경

## 3. phase_change 시 room.status 전환

- [x] 3.1 `gameStore.ts` `phase_change` 핸들러에서 `set(updates)` → `set((s) => ({ ...updates, room: s.room ? { ...s.room, status: 'playing' } : s.room }))`로 변경 (startTimer 호출은 set 이후 별도로 유지)

## 4. ChatLog player_name 활용

- [x] 4.1 `frontend/src/components/ChatLog.tsx`의 `getPlayerName` 함수에서 `msg.player_name`이 있으면 바로 반환, 없으면 기존 room.players 조회 폴백

## 5. listRooms 응답 타입 수정

- [x] 5.1 `frontend/src/api.ts`의 `listRooms`를 `request<import('./types').Room[]>('/rooms')`로 반환 타입 변경
- [x] 5.2 `frontend/src/pages/LobbyPage.tsx`의 `fetchRooms`에서 `data.rooms ?? []` → `data ?? []`로 변경

## 6. createRoom 요청/응답 수정

- [x] 6.1 `frontend/src/api.ts`의 `request` 함수에서 `{ headers: {...}, ...options }` → `const { headers: optHeaders, ...restOptions } = options ?? {}`로 분리하여 헤더 병합 버그 수정 (`...options` spread가 headers 키를 덮어쓰는 문제)
- [x] 6.2 `frontend/src/api.ts`의 `CreateRoomParams`에 `game_type?: string`과 `max_humans?: number` optional 필드 추가 (`player_name`은 헤더 전달에도 필요하므로 유지)
- [x] 6.3 `frontend/src/api.ts`의 `createRoom` 함수에서 body를 `{ name, visibility, game_type: params.game_type ?? 'mafia', max_humans: params.max_humans ?? 1 }`로 교체하고 `headers: { 'X-Player-Name': params.player_name }` 추가
- [x] 6.4 `frontend/src/api.ts`의 `JoinResponse`에서 `room_id: string` → `id: string`으로 필드명 변경
- [x] 6.5 `frontend/src/pages/LobbyPage.tsx`의 `handleCreateRoom`에서 응답의 `res.room_id` → `res.id` 변경 (createRoom 호출부의 player_name은 그대로 유지)
- [x] 6.6 `frontend/src/pages/LobbyPage.tsx`의 `handleJoinByCode`에서 `res.room_id` → `res.id` 변경 (`handleJoinRoom`은 `joiningRoom.id`를 직접 사용하므로 변경 불필요)

## 7. 빌드 및 타입 검증

- [x] 7.1 `cd frontend && npx tsc --noEmit`으로 TypeScript 타입 오류 없음 확인
