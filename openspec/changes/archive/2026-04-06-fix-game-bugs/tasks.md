## 1. Fix EventGameOver race (game_manager.go)

- [x] 1.1 In `GameManager.start()`, locate the game goroutine where `game.Start(gameCtx)` is called and `cancelGame()` is deferred
- [x] 1.2 Remove the `defer cancelGame()` and instead drain the event channel after `game.Start()` returns: loop `select { case event := <-game.Subscribe(): relay it; default: break }`, then call `cancelGame()` explicitly
- [x] 1.3 Verify the relay logic inside the drain loop uses the same `gm.GameEventFunc` call as the event goroutine

## 2. Fix activeGames memory leak (game_manager.go)

- [x] 2.1 In the game goroutine's cleanup block (after `game.Start()` returns and all cleanup is done), add `gm.mu.Lock(); delete(gm.activeGames, room.ID); gm.mu.Unlock()`
- [x] 2.2 Confirm no other code path removes from `activeGames` in a way that could cause a double-delete (check `TryRecover`, `RecoverGame`)

## 3. Fix AI goroutine hang (agent.go)

- [x] 3.1 In `decideVote`, replace `a.outCh <- AgentOutput{...}` with `select { case a.outCh <- AgentOutput{...}: case <-ctx.Done(): return }`
- [x] 3.2 Apply the same pattern to `decideKill`
- [x] 3.3 Apply the same pattern to `decideInvestigate`
- [x] 3.4 Confirm `delayedOutput()` already uses this pattern (no change needed there)

## 4. Fix checkpoint shallow copy (phases.go)

- [x] 4.1 In `save()`, check the type of the checkpoint's player field — if it is `[]*entity.Player`, change it to `[]entity.Player` in the relevant struct definition
- [x] 4.2 Rewrite the player copy in `save()` to deep-copy: `players := make([]entity.Player, len(pm.state.Players)); for i, p := range pm.state.Players { players[i] = *p }; snapshot.Players = players`
- [x] 4.3 Update any code that reads the checkpoint and expects `[]*entity.Player` to work with the new `[]entity.Player` type (check crash-recovery path in `game_manager.go` or `phases.go`)

## 5. Fix validation inversion in RecordMafiaKill and RecordVote (phases.go)

- [x] 5.1 Rewrite `RecordMafiaKill` to find the killer first (`var killer *entity.Player; for _, p := range ... { if p.ID == killerID { killer = p; break } }`), then guard with `if killer == nil || killer.Role != entity.RoleMafia || !killer.IsAlive { return }`
- [x] 5.2 Rewrite `RecordVote` with the same find-then-guard pattern: find voter by ID, guard with `if voter == nil || !voter.IsAlive { return }`
- [x] 5.3 Run `go test ./internal/games/mafia/...` and confirm all existing tests pass (especially `TestRecordMafiaKill_*` and `TestRecordVote_*`)

## 6. Final verification

- [x] 6.1 Run `go build ./...` — no compile errors
- [x] 6.2 Run `go vet ./...` — no vet warnings
- [x] 6.3 Run `go test ./...` — all tests pass
