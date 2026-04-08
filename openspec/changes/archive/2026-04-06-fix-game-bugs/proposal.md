## Why

The Go backend contains several correctness bugs — race conditions, memory leaks, and goroutine leaks — that silently corrupt game state or prevent proper cleanup. The most critical bug causes roughly 50% of game-over events to be silently dropped, meaning players never see the winner announcement in half of all games.

## What Changes

- Fix `EventGameOver` race condition in `GameManager.start()` so the game-over event is always relayed before the event goroutine exits
- Add `activeGames` map cleanup when a game ends normally (currently only cleaned on recovery/restart)
- Fix AI agent goroutine leak: `decideVote`, `decideKill`, and `decideInvestigate` use blocking channel sends that ignore context cancellation
- Fix shallow copy in `PhaseManager.save()` so checkpoint player states are independent of live game state
- Fix validation logic inversion in `RecordMafiaKill` and `RecordVote` that lets unrecognized player IDs bypass the validation guard

## Capabilities

### New Capabilities

None — this is a pure bug-fix change with no new user-facing capabilities.

### Modified Capabilities

- `game-lifecycle`: Game start, end, and cleanup flow changes (EventGameOver delivery guarantee, activeGames cleanup)
- `ai-agent`: AI agent output-send pattern changes (ctx-aware channel send)
- `phase-manager`: Checkpoint save and vote/kill recording logic changes

## Impact

- `internal/platform/game_manager.go`: EventGameOver race fix, activeGames delete on end
- `internal/ai/agent.go`: `decideVote`, `decideKill`, `decideInvestigate` send pattern
- `internal/games/mafia/phases.go`: `save()` deep copy, `RecordMafiaKill`/`RecordVote` validation
- No API surface changes, no schema changes, no dependency changes
- Existing tests in `phases_test.go` remain valid; no test changes required for the logic fixes
