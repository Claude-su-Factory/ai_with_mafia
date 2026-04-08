## 1. 방 목록 인원수 0 수정

- [x] 1.1 `internal/platform/room.go`의 `ListPublic()`에서 DB 결과를 순회할 때, `s.rooms[room.ID]`가 존재하면 메모리 버전으로 교체하는 merge 로직 추가

## 2. 1인 게임 시작 허용

- [x] 2.1 `frontend/src/components/WaitingRoom.tsx` 40번째 줄 `canStart` 조건을 `room.players.length >= 2`에서 `room.players.length >= 1`로 수정
- [x] 2.2 버튼 비활성 메시지 `최소 2명 필요 (현재 ${room.players.length}명)`를 제거하고, 비활성 상태 없이 항상 `게임 시작` 텍스트 표시

## 3. max_humans 기본값 수정

- [x] 3.1 `frontend/src/api.ts`의 `createRoom`에서 `max_humans: params.max_humans ?? 1`을 `max_humans: params.max_humans ?? 6`으로 변경

## 4. 검증

- [ ] 4.1 방 생성 후 로비 목록에서 1명으로 표시되는지 확인 (수동)
- [ ] 4.2 혼자서 게임 시작 버튼이 활성화되는지 확인 (수동)
- [ ] 4.3 다른 계정으로 같은 방 참가 시 room is full 에러 없이 입장되는지 확인 (수동)
