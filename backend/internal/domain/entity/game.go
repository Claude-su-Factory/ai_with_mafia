package entity

import "context"

type Phase string

const (
	PhaseDayDiscussion Phase = "day_discussion"
	PhaseDayVote       Phase = "day_vote"
	PhaseNight         Phase = "night"
	PhaseResult        Phase = "result"
)

type GameEventType string

const (
	EventChat          GameEventType = "chat"
	EventPhaseChange   GameEventType = "phase_change"
	EventVote          GameEventType = "vote"
	EventKill          GameEventType = "kill"
	EventNightAction   GameEventType = "night_action"
	EventPlayerLeft    GameEventType = "player_left"
	EventPlayerReplaced GameEventType = "player_replaced"
	EventGameOver      GameEventType = "game_over"
	EventMafiaChat     GameEventType = "mafia_chat"
	EventTimerUpdate   GameEventType = "timer_update"
	EventMafiaChannelOpen GameEventType = "mafia_channel_open"
)

type GameEvent struct {
	Type      GameEventType
	PlayerID  string
	Payload   map[string]any
	// MafiaOnly: true이면 마피아 플레이어에게만 전달
	MafiaOnly bool
}

type GameState struct {
	Phase                 Phase
	AlivePlayers          []string
	Votes                 map[string]string // voterID -> targetID
	Round                 int
	WinnerTeam            string // "mafia" | "citizen" | ""
	TimerRemainingSeconds int
}

type Game interface {
	Start(ctx context.Context)
	HandleAction(playerID string, action Action) error
	State() GameState
	Subscribe() <-chan GameEvent
}

type Action struct {
	Type    string
	Payload map[string]any
}
