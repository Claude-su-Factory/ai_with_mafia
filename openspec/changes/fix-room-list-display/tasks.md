## 1. ListPublic() — 메모리에 없는 방 제외

- [x] 1.1 `internal/platform/room.go`의 `ListPublic()`에서 DB 결과 merge 루프를 수정: 메모리에 존재하는 방만 result에 포함하고, 메모리에 없는 방은 제외

## 2. ToRoomResponse() — AI 플레이어 제외

- [x] 2.1 `internal/platform/room.go`의 `ToRoomResponse()`에서 players 순회 시 `p.IsAI`가 true인 플레이어는 건너뜀

## 3. 빌드 및 검증

- [x] 3.1 `go build ./...` 컴파일 오류 없음 확인
- [ ] 3.2 방장이 대기실 나가면 방 목록에서 사라지는지 확인 (수동)
- [ ] 3.3 게임 진행 중 방 목록의 인원수가 AI 제외 실제 사용자 수로 표시되는지 확인 (수동)
