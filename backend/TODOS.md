# Backend TODOs

## TODO-1: Mafia kill no-consensus feedback event

**What:** When the night phase ends with no mafia consensus (mafia voted for different targets), emit a `night_result` event telling clients "the mafia could not agree — no one was killed tonight."

**Why:** Currently `processMafiaKill` silently does nothing on no-consensus. Players see the phase transition but get no explanation. This is a dead moment in the game — bad UX, especially confusing on the first round when new players don't understand the consensus rule.

**Pros:** Clears up a confusing silent game state. Reinforces the mafia consensus mechanic by surfacing it to all players.

**Cons:** Reveals to citizens that the mafia was split (minor strategic info leak — acceptable since they don't know who voted for whom).

**Context:** `processMafiaKill` in `internal/games/mafia/phases.go` already has a "no consensus" path at the end of the function — it just falls through without emitting. Add an `EventNightAction` or `EventKill` with reason `"no_consensus"` there.

**Depends on:** Nothing blocked.

---

## TODO-2: Redis pub/sub reconnect resilience

**What:** The Redis pub/sub subscriber in `internal/platform/ws/hub.go` (`startSubscriber`) exits silently when the channel closes (`!ok` branch) or when `ctx.Done()` fires. There is no reconnect loop.

**Why:** If Redis drops the connection (network blip, Redis restart), the subscriber goroutine exits and all instances stop relaying cross-instance WS messages. Players on other instances stop seeing events — game effectively becomes single-instance until server restart.

**Pros:** Surviving a Redis blip without a full server restart. Critical for any multi-instance prod deployment.

**Cons:** Small added complexity — a reconnect loop with exponential backoff (~10 lines).

**Context:** `startSubscriber` in `internal/platform/ws/hub.go` uses `rdb.PSubscribe(ctx, "room:*")`. When the subscribe channel closes, wrap the outer loop with a reconnect delay (`time.Sleep(backoff)` up to ~30s) and call `PSubscribe` again. Only exit on `ctx.Done()`.

**Depends on:** Nothing blocked.
