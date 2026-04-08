## MODIFIED Requirements

### Requirement: EventGameOver is always delivered to clients

The system SHALL guarantee that the `EventGameOver` event emitted by a finished game is relayed to all connected clients before the event relay goroutine exits.

#### Scenario: Game ends, event goroutine sees both channels ready simultaneously
- **WHEN** `game.Start()` returns (game over) and `cancelGame()` is about to be called
- **THEN** the game goroutine MUST drain any remaining events from the event channel and relay them before calling `cancelGame()`, so clients always receive `EventGameOver`

#### Scenario: Game ends with no pending events
- **WHEN** `game.Start()` returns and the event channel is empty
- **THEN** the drain loop exits immediately (no-op) and `cancelGame()` is called normally

### Requirement: activeGames map is cleaned up on normal game end

The system SHALL remove a game's entry from the `activeGames` map when that game completes normally, preventing unbounded memory growth.

#### Scenario: Game reaches a win condition
- **WHEN** `game.Start()` returns because a winner was found
- **THEN** `delete(gm.activeGames, room.ID)` MUST be called under the manager lock before the game goroutine exits

#### Scenario: Server restart after many completed games
- **WHEN** the server has hosted N completed games since startup
- **THEN** the `activeGames` map MUST contain only currently-running games, not completed ones
