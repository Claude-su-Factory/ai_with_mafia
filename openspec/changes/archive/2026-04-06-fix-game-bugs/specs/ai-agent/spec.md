## MODIFIED Requirements

### Requirement: AI agent action functions respect context cancellation

The system SHALL ensure that all AI agent output sends (vote, kill, investigate decisions) use a context-aware select pattern so the goroutine exits cleanly when the game context is cancelled.

#### Scenario: outCh is full and game context is cancelled
- **WHEN** `decideVote`, `decideKill`, or `decideInvestigate` attempts to send to `outCh` AND `outCh` is full AND `ctx.Done()` fires
- **THEN** the function MUST return immediately without blocking, preventing a goroutine leak

#### Scenario: outCh has space, action is sent normally
- **WHEN** `decideVote`, `decideKill`, or `decideInvestigate` attempts to send to `outCh` AND `outCh` has capacity
- **THEN** the action MUST be sent successfully and the function returns normally

#### Scenario: Context already cancelled before send
- **WHEN** `ctx.Done()` has already fired before the send is attempted
- **THEN** the function MUST return without sending (same as the concurrent cancellation case)
