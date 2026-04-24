package ws

// buildAbortedGameOverPayload returns a game_over payload shaped like the
// normal completion path (winner / round / duration_sec / players) so the
// frontend GameOverResult consumer never sees undefined fields.
//
// It is used when the game ends because every human left mid-game — no
// canonical winner exists, so the winner field carries the sentinel "aborted"
// and the "reason" field explains the cause for UI routing.
func buildAbortedGameOverPayload() map[string]any {
	return map[string]any{
		"winner":       "aborted",
		"round":        0,
		"duration_sec": 0,
		"players":      []map[string]any{}, // non-nil so JSON serialises as []
		"reason":       "all_humans_left",
	}
}
