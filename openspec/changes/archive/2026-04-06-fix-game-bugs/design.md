## Context

The backend is a Go service using Fiber v2 for HTTP/WebSocket and a `GameManager` that orchestrates mafia games. Each game runs in two goroutines: a game goroutine (calls `game.Start()`, then defers `cancelGame()`) and an event goroutine (reads from `game.Subscribe()` and relays events to clients).

Five bugs were found across three files during a correctness audit:

1. **EventGameOver race** (`game_manager.go`): When `game.Start()` returns (game over), the game goroutine calls `cancelGame()` via defer. At this moment both the event channel (which just received `EventGameOver`) and `gameCtx.Done()` are simultaneously ready in the event goroutine's `select`. Go picks randomly — the game-over event is dropped ~50% of the time.

2. **activeGames leak** (`game_manager.go`): `delete(gm.activeGames, room.ID)` is only called in `TryRecover` and `RecoverGame`, never when a game ends normally. The map grows unboundedly.

3. **AI goroutine hang** (`agent.go`): `decideVote`, `decideKill`, `decideInvestigate` send to `a.outCh` with a direct `a.outCh <- out` (blocking). If `outCh` is full and the context is cancelled, the goroutine leaks.

4. **Shallow copy in checkpoint** (`phases.go`): `save()` copies `[]*entity.Player` by pointer, so checkpoint data shares the same `Player` structs as the live game.

5. **Validation inversion** (`phases.go`): `RecordMafiaKill` and `RecordVote` guard with `return` only when the player IS found AND fails a condition. If the player is NOT found, the guard never fires, and the kill/vote is recorded anyway.

## Goals / Non-Goals

**Goals:**
- Fix all 5 bugs with minimal, targeted changes
- Preserve existing behaviour for the happy path (no behaviour regressions)
- Keep changes reviewable — one logical fix per bug

**Non-Goals:**
- Refactoring unrelated code
- Adding new features or changing the game protocol
- Changing test files (existing tests should pass as-is)

## Decisions

### Bug 1 — EventGameOver: drain before cancel

**Decision**: After `game.Start()` returns, drain the event channel and relay any pending events before calling `cancelGame()`.

```go
// In the game goroutine, after game.Start(gameCtx):
for {
    select {
    case event := <-game.Subscribe():
        if gm.GameEventFunc != nil {
            gm.GameEventFunc(room.ID, event)
        }
    default:
        goto drained
    }
}
drained:
cancelGame()
```

**Alternative considered**: Increase the event channel buffer. Rejected — doesn't fix the race, only reduces its frequency.

**Alternative considered**: Close the event channel from `game.go` after sending EventGameOver and use `range` in the relay goroutine. Rejected — more invasive, requires protocol change.

### Bug 2 — activeGames: delete on end

**Decision**: Add `delete(gm.activeGames, room.ID)` (under `gm.mu.Lock()`) in the game goroutine's cleanup block, immediately before returning.

### Bug 3 — AI channel send: use select with ctx.Done()

**Decision**: Replace all direct `a.outCh <- out` sends in agent action functions with:

```go
select {
case a.outCh <- out:
case <-ctx.Done():
    return
}
```

This matches the pattern already used in `delayedOutput()`.

### Bug 4 — Checkpoint: deep copy Players

**Decision**: In `save()`, construct a new `[]entity.Player` (value slice) from the pointer slice, copying each struct by value:

```go
func (pm *PhaseManager) save() {
    players := make([]entity.Player, len(pm.state.Players))
    for i, p := range pm.state.Players {
        players[i] = *p  // dereference — copy struct value
    }
    // store players (as values) in snapshot
}
```

**Note**: If `GameState.checkpoint` currently stores `[]*entity.Player`, the type must change to `[]entity.Player` for this to be meaningful. Verify and align.

### Bug 5 — Validation: invert the guard

**Decision**: Rewrite the guard to find the player first, then check conditions:

```go
func (pm *PhaseManager) RecordMafiaKill(killerID, targetID string) {
    var killer *entity.Player
    for _, p := range pm.state.Players {
        if p.ID == killerID {
            killer = p
            break
        }
    }
    if killer == nil || killer.Role != entity.RoleMafia || !killer.IsAlive {
        return
    }
    pm.state.NightKills[killerID] = targetID
}
```

Apply the same pattern to `RecordVote`.

## Risks / Trade-offs

- **Bug 1 drain loop**: The drain uses a `default` branch so it is non-blocking and terminates immediately after the channel is empty. Risk of infinite loop if the game somehow keeps producing events after `Start()` returns — mitigated by the fact that `game.Start()` only returns after `endGame()` sends exactly one `EventGameOver` and then no more writes occur.

- **Bug 4 checkpoint type change**: If the snapshot struct changes from `[]*entity.Player` to `[]entity.Player`, any code reading the checkpoint to reconstruct live `*entity.Player` pointers must be reviewed. Low risk — checkpoint is only used for crash recovery.

## Migration Plan

1. Apply all 5 fixes in a single PR (they are independent, non-overlapping changes)
2. Run existing test suite — no test changes needed
3. Deploy; no migration steps required (no schema changes, no config changes)
4. Rollback: revert the PR — stateless changes, no data migration needed
