package dto

type ChatMessageDTO struct {
	PlayerID   string `json:"player_id"`
	PlayerName string `json:"player_name"`
	Message    string `json:"message"`
	MafiaOnly  bool   `json:"mafia_only,omitempty"`
}

type VoteRequest struct {
	TargetID string `json:"target_id"`
}

type NightActionRequest struct {
	ActionType string `json:"action_type"` // "kill" | "investigate"
	TargetID   string `json:"target_id"`
}

type ActionRequest struct {
	Type    string         `json:"type"`
	Chat    *ChatMessageDTO    `json:"chat,omitempty"`
	Vote    *VoteRequest       `json:"vote,omitempty"`
	Night   *NightActionRequest `json:"night,omitempty"`
}

type GameEventDTO struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type GameStateDTO struct {
	Phase        string      `json:"phase"`
	AlivePlayers []PlayerDTO `json:"alive_players"`
	Round        int         `json:"round"`
	TimerSeconds int         `json:"timer_seconds"`
}

// GameSnapshot is a point-in-time view of active game state shared between
// the game manager and the WebSocket hub.
type GameSnapshot struct {
	Phase             string
	Round             int
	TimerRemainingSec int
	AlivePlayerIDs    []string
	Votes             map[string]string // voterID → targetID, nil if not vote phase
}
