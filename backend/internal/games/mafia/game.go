package mafia

import (
	"context"
	"time"

	"go.uber.org/zap"

	"ai-playground/internal/domain/entity"
)

const GameName = "mafia"

// MafiaGame implements entity.Game.
type MafiaGame struct {
	room      *entity.Room
	state     *GameState
	phase     *PhaseManager
	eventCh   chan entity.GameEvent
	timers    Timers
	logger    *zap.Logger
	startedAt time.Time
}

func newGame(room *entity.Room, timers Timers, logger *zap.Logger) *MafiaGame {
	eventCh := make(chan entity.GameEvent, 64)
	players := room.GetPlayers()

	AssignRoles(players)

	state := NewGameState(players)
	pm := NewPhaseManager(state, eventCh, timers, logger, room.ID)

	return &MafiaGame{
		room:    room,
		state:   state,
		phase:   pm,
		eventCh: eventCh,
		timers:  timers,
		logger:  logger,
	}
}

func (g *MafiaGame) Start(ctx context.Context) {
	g.startedAt = time.Now()
	g.state.mu.Lock()
	g.state.Round = 1
	g.state.mu.Unlock()

	for {
		// 낮 토론
		g.phase.RunDayDiscussion(ctx)
		if ctx.Err() != nil {
			return
		}

		// 투표
		g.phase.RunDayVote(ctx)
		if ctx.Err() != nil {
			return
		}

		if winner := g.phase.CheckWin(); winner != "" {
			g.endGame(winner)
			return
		}

		// 밤
		g.phase.RunNight(ctx)
		if ctx.Err() != nil {
			return
		}

		if winner := g.phase.CheckWin(); winner != "" {
			g.endGame(winner)
			return
		}

		g.state.mu.Lock()
		g.state.Round++
		g.state.mu.Unlock()
	}
}

func (g *MafiaGame) HandleAction(playerID string, action entity.Action) error {
	switch action.Type {
	case "chat":
		msg, _ := action.Payload["message"].(string)
		mafiaOnly, _ := action.Payload["mafia_only"].(bool)
		evType := entity.EventChat
		if mafiaOnly {
			evType = entity.EventMafiaChat
		}
		senderName := ""
		if p := g.room.PlayerByID(playerID); p != nil {
			senderName = p.Name
		}
		select {
		case g.eventCh <- entity.GameEvent{
			Type:      evType,
			PlayerID:  playerID,
			MafiaOnly: mafiaOnly,
			Payload: map[string]any{
				"sender_id":   playerID,
				"sender_name": senderName,
				"message":     msg,
			},
		}:
		default:
			g.logger.Warn("event channel full, dropping chat event",
				zap.String("room_id", g.room.ID),
				zap.String("event_type", string(evType)))
		}

	case "vote":
		targetID, _ := action.Payload["target_id"].(string)
		g.phase.RecordVote(playerID, targetID)

	case "kill":
		targetID, _ := action.Payload["target_id"].(string)
		g.phase.RecordMafiaKill(playerID, targetID)

	case "investigate":
		targetID, _ := action.Payload["target_id"].(string)
		g.phase.RecordInvestigation(playerID, targetID)
	}
	return nil
}

func (g *MafiaGame) State() entity.GameState {
	return g.phase.State()
}

func (g *MafiaGame) Subscribe() <-chan entity.GameEvent {
	return g.eventCh
}

// SetOnSave wires a checkpoint callback through the phase manager.
// fn receives a safe snapshot of the current game state.
func (g *MafiaGame) SetOnSave(fn func(ctx context.Context, data CheckpointData) error) {
	g.phase.SetOnSave(fn)
}

// NewGame creates a new MafiaGame instance.
func NewGame(room *entity.Room, timers Timers, logger *zap.Logger) *MafiaGame {
	return newGame(room, timers, logger)
}

func (g *MafiaGame) endGame(winner string) {
	g.state.mu.Lock()
	g.state.WinnerTeam = winner
	g.state.Phase = entity.PhaseResult
	round := g.state.Round
	g.state.mu.Unlock()

	durationSec := int(time.Since(g.startedAt).Seconds())

	players := g.room.GetPlayers()
	playerList := make([]map[string]any, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, map[string]any{
			"id":       p.ID,
			"name":     p.Name,
			"role":     string(p.Role),
			"is_ai":    p.IsAI,
			"survived": p.IsAlive,
		})
	}

	select {
	case g.eventCh <- entity.GameEvent{
		Type: entity.EventGameOver,
		Payload: map[string]any{
			"winner":       winner,
			"round":        round,
			"duration_sec": durationSec,
			"players":      playerList,
		},
	}:
	default:
		g.logger.Warn("event channel full, dropping game_over event", zap.String("room_id", g.room.ID))
	}
}

