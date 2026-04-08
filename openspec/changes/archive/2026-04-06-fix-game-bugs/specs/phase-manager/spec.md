## MODIFIED Requirements

### Requirement: Checkpoint saves independent copies of player state

The system SHALL store a deep copy of player structs in game checkpoints so that subsequent mutations to live player state do not corrupt previously saved checkpoint data.

#### Scenario: Player state changes after checkpoint is saved
- **WHEN** `save()` records a checkpoint AND a player's `IsAlive` or `Role` field is later modified in the live game
- **THEN** the saved checkpoint MUST still reflect the player state at the time `save()` was called, not the modified state

#### Scenario: Crash recovery uses checkpoint data
- **WHEN** the server restarts and reads a checkpoint
- **THEN** the recovered player states MUST match the state at checkpoint time, unaffected by any modifications that occurred after the save

### Requirement: RecordMafiaKill validates the killer before recording

The system SHALL reject kill registrations where the killer player does not exist, is not alive, or does not have the Mafia role.

#### Scenario: Kill from a non-existent player ID
- **WHEN** `RecordMafiaKill` is called with a `killerID` that is not in `pm.state.Players`
- **THEN** the kill MUST be silently ignored (not recorded in `NightKills`)

#### Scenario: Kill from an alive Mafia player
- **WHEN** `RecordMafiaKill` is called with a valid alive Mafia player as killer
- **THEN** the kill MUST be recorded in `NightKills`

#### Scenario: Kill from a dead player
- **WHEN** `RecordMafiaKill` is called with a player who is dead (`IsAlive == false`)
- **THEN** the kill MUST be silently ignored

### Requirement: RecordVote validates the voter before recording

The system SHALL reject vote registrations where the voter player does not exist or is not alive.

#### Scenario: Vote from a non-existent player ID
- **WHEN** `RecordVote` is called with a `voterID` that is not in `pm.state.Players`
- **THEN** the vote MUST be silently ignored (not recorded in `Votes`)

#### Scenario: Vote from an alive player
- **WHEN** `RecordVote` is called with a valid alive player as voter
- **THEN** the vote MUST be recorded in `Votes`

#### Scenario: Vote from a dead player
- **WHEN** `RecordVote` is called with a player who is dead (`IsAlive == false`)
- **THEN** the vote MUST be silently ignored
