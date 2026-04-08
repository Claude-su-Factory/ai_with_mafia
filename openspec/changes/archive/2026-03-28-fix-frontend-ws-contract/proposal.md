## Why

서버 WS 계약 수정(fix-server-ws-contract) 이후 프론트엔드가 새 이벤트 구조를 처리하지 못해 채팅이 렌더링되지 않고, API 계약 불일치로 방 생성과 방 목록이 동작하지 않는다. 게임이 시작돼도 화면이 전환되지 않는 렌더링 버그도 포함된다.

## What Changes

- **chat / mafia_chat 핸들러 수정**: `event.player_id`, `event.message` flat 접근 → `event.payload.sender_id`, `event.payload.sender_name`, `event.payload.message` payload 접근으로 변경
- **WsEvent 타입 수정**: `types.ts`의 chat/mafia_chat union 타입을 payload wrapper 구조로 변경
- **ChatLog 이름 표시 개선**: `sender_name`을 `ChatMessage.player_name`에 저장해 room.players 조회 없이 직접 표시
- **phase_change 시 room.status 전환**: phase_change 핸들러에서 `room.status`를 `'playing'`으로 업데이트해 WaitingRoom → GameRoom 전환
- **createRoom 요청 수정**: `game_type`, `max_humans` body 추가 + `X-Player-Name` 헤더로 플레이어 이름 전달; 응답에서 `res.room_id` → `res.id`
- **listRooms 응답 타입 수정**: 서버가 배열을 직접 반환하므로 `{ rooms: [] }` 래핑 제거

## Capabilities

### New Capabilities

없음.

### Modified Capabilities

- `game-ui`: chat/mafia_chat 이벤트 payload 구조 변경 수신, phase_change 시 게임 화면 전환 요구사항 추가
- `lobby-ui`: 방 생성 요청 필드 및 응답 처리, 방 목록 응답 shape 요구사항 변경

## Impact

- `frontend/src/types.ts`: WsEvent chat/mafia_chat union 타입
- `frontend/src/store/gameStore.ts`: chat, mafia_chat, phase_change 핸들러
- `frontend/src/api.ts`: createRoom 요청/응답 타입, listRooms 반환 타입
- `frontend/src/pages/LobbyPage.tsx`: createRoom 호출부, listRooms 응답 처리
- `frontend/src/components/ChatLog.tsx`: player_name 직접 표시
