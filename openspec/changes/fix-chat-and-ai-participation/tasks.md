## 1. 채팅 payload 구조 수정

- [x] 1.1 `frontend/src/components/ChatInput.tsx`의 `handleSend()`에서 `sendAction('chat', { message: trimmed })`를 `sendAction('chat', { chat: { message: trimmed } })`로 변경

## 2. AI 낮 토론 첫 발언 추가

- [x] 2.1 `backend/internal/ai/agent.go`의 `onPhaseChange()`에 `case entity.PhaseDayDiscussion:` 추가
- [x] 2.2 해당 케이스에서 `callLLM()`을 호출하여 게임 시작 상황에 맞는 첫 발언 생성 ("낮 토론이 시작됐습니다. 플레이어들을 관찰하며 자연스럽게 대화를 시작하세요." 프롬프트)
- [x] 2.3 생성된 발언을 `delayedOutput()`으로 전송 (기존 채팅 응답과 동일한 패턴)

## 3. 빈 방 DB 삭제 누락 수정

- [x] 3.1 `backend/internal/platform/room.go`의 `RemovePlayer()`에서 `room.HumanCount() == 0`으로 메모리에서 방을 삭제할 때 `s.roomRepo.Delete(context.Background(), roomID)` 호출 추가
- [x] 3.2 `RoomService.Delete()`에도 동일하게 `s.roomRepo.Delete()` 호출 추가 (일관성)

## 4. Rate limit 완화

- [x] 4.1 `backend/internal/platform/ws/hub.go`의 `msgRateLimit` 상수를 `500 * time.Millisecond`에서 `200 * time.Millisecond`로 변경

## 5. 빌드 및 검증

- [x] 5.1 `go build ./...` 컴파일 오류 없음 확인
- [ ] 5.2 채팅 전송 시 WS 클라이언트에 broadcast되는지 확인 (수동)
- [ ] 5.3 게임 시작 후 낮 토론 페이즈에 AI 에이전트가 발언하는지 확인 (수동)
- [ ] 5.4 방장이 대기실 나가면 방이 목록에서 사라지는지 확인 (수동)
