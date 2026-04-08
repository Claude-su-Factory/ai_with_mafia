## 1. night_action 경찰에게만 전송

- [x] 1.1 `internal/games/mafia/phases.go`의 `RecordInvestigation`에서 `night_action` emit 시 `PlayerID: policeID` 필드 설정 (`GameEvent.PlayerID`는 이미 entity에 정의되어 있음)
- [x] 1.2 `cmd/server/main.go`의 `gameEventFunc`에서 `event.PlayerID != ""`인 경우 `hub.SendToPlayer(roomID, event.PlayerID, payload)`로 라우팅 (현재는 항상 Broadcast만 호출)

## 2. phase_change에 round 필드 추가

- [x] 2.1 `RunDayDiscussion` phase_change emit payload에 `"round": pm.state.Round` 추가
- [x] 2.2 `RunDayVote` phase_change emit payload에 `"round": pm.state.Round` 추가
- [x] 2.3 `RunNight` MafiaOnly phase_change emit payload에 `"round": pm.state.Round` 추가
- [x] 2.4 `RunNight` 전체 대상 phase_change emit payload에 `"round": pm.state.Round` 추가 (RunNight는 phase_change를 두 번 emit함)

## 3. AI 채팅 payload 구조 통일

- [x] 3.1 `internal/ai/agent.go`의 `AgentOutput` 구조체에 `PlayerName string` 필드 추가
- [x] 3.2 `agent.go`의 chat `AgentOutput` 생성 시 `PlayerName: a.Persona.Name` 포함
- [x] 3.3 `internal/ai/manager.go`의 `SetCallbacks` broadcast 시그니처에 `playerName string` 파라미터 추가, `handleOutput`에서 `out.PlayerName` 전달
- [x] 3.4 `cmd/server/main.go` AI chat callback을 `func(roomID, playerID, playerName, message string, mafiaOnly bool)`로 변경하고, `mafiaOnly`에 따라 `Type`을 `entity.EventChat` / `entity.EventMafiaChat`으로 분기하여 `gm.gameEventFunc`로 전달 (플레이어 채팅과 동일한 타입 규칙)

## 4. 빌드 및 검증

- [x] 4.1 `go build ./...`로 컴파일 오류 없음 확인
- [x] 4.2 서버 실행 후 플레이어 채팅/AI 채팅 payload 구조가 동일한지 로그로 확인
- [x] 4.3 경찰 조사 결과가 경찰 클라이언트에만 전달되는지 확인
- [x] 4.4 phase_change 이벤트에 round 필드가 포함되는지 확인
