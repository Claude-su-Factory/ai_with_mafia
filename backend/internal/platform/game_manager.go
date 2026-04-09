package platform

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"go.uber.org/zap"

	"ai-playground/config"
	"ai-playground/internal/ai"
	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
	"ai-playground/internal/games/mafia"
	"ai-playground/internal/repository"
)

// GameManager coordinates game lifecycle: leader election, AI spawn, game start/recovery,
// snapshot delivery, and event routing.
type GameManager struct {
	mu             sync.Mutex
	mafiaCfg       *config.MafiaGameConfig
	ai             *ai.Manager
	personaPool    *ai.PersonaPool
	leaderLock     *LeaderLock
	instanceID     string
	gameStateRepo  *repository.GameStateRepository
	aiHistoryRepo  *repository.AIHistoryRepository
	gameResultRepo *repository.GameResultRepository
	roomSvc        *RoomService
	logger         *zap.Logger
	activeGames    map[string]activeGame // roomID -> game + cancel

	// Callbacks wired by the caller after construction.
	GameEventFunc  func(roomID string, event entity.GameEvent)
	UpdateRoleFunc func(roomID, playerID string, role entity.Role)
}

type activeGame struct {
	game   entity.Game
	cancel context.CancelFunc
	ctx    context.Context // 신규
}

func NewGameManager(
	mafiaCfg *config.MafiaGameConfig,
	a *ai.Manager,
	pool *ai.PersonaPool,
	ll *LeaderLock,
	instanceID string,
	gsRepo *repository.GameStateRepository,
	ahRepo *repository.AIHistoryRepository,
	grRepo *repository.GameResultRepository,
	roomSvc *RoomService,
	l *zap.Logger,
) *GameManager {
	return &GameManager{
		mafiaCfg:       mafiaCfg,
		ai:             a,
		personaPool:    pool,
		leaderLock:     ll,
		instanceID:     instanceID,
		gameStateRepo:  gsRepo,
		aiHistoryRepo:  ahRepo,
		gameResultRepo: grRepo,
		roomSvc:        roomSvc,
		logger:         l,
		activeGames:    make(map[string]activeGame),
	}
}

func (gm *GameManager) StartGame(ctx context.Context, room *entity.Room) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	return gm.start(ctx, room, nil)
}

func (gm *GameManager) RestartGame(ctx context.Context, room *entity.Room) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	if ag, ok := gm.activeGames[room.ID]; ok {
		ag.cancel()
		delete(gm.activeGames, room.ID)
	}
	return gm.start(ctx, room, nil)
}

// RecoverGame restarts a game loop using saved state (after instance crash).
func (gm *GameManager) RecoverGame(ctx context.Context, room *entity.Room, histories map[string][]anthropic.MessageParam) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	return gm.start(ctx, room, histories)
}

func (gm *GameManager) start(parent context.Context, room *entity.Room, preloadedHistories map[string][]anthropic.MessageParam) error {
	// Acquire leader lock — abort if another instance already leads
	if gm.leaderLock != nil {
		if !gm.leaderLock.Acquire(parent, room.ID, gm.instanceID) {
			gm.logger.Info("leader lock not acquired, skipping",
				zap.String("room_id", room.ID),
				zap.String("instance_id", gm.instanceID))
			return nil
		}
	}

	// Add AI players to fill up to TotalPlayers (skip if already full, e.g. recovery path)
	aiCount := mafia.TotalPlayers - len(room.GetPlayers())
	personas := gm.personaPool.Assign(mafia.TotalPlayers) // always assign full pool for SpawnAgents
	if aiCount > 0 {
		for i := range aiCount {
			id := fmt.Sprintf("ai-%s-%d", room.ID, i)
			p := entity.NewPlayer(id, personas[i].Name, true)
			room.AddPlayer(p)
		}
	}

	timers := mafia.Timers{
		DayDiscussion: time.Duration(gm.mafiaCfg.Timers.DayDiscussion) * time.Second,
		DayVote:       time.Duration(gm.mafiaCfg.Timers.DayVote) * time.Second,
		Night:         time.Duration(gm.mafiaCfg.Timers.Night) * time.Second,
	}
	game := mafia.NewGame(room, timers, gm.logger)
	room.SetStatus(entity.RoomStatusPlaying)

	// Sync roles to WS clients after role assignment
	if gm.UpdateRoleFunc != nil {
		for _, p := range room.GetPlayers() {
			gm.UpdateRoleFunc(room.ID, p.ID, p.Role)
		}
	}

	gameCtx, cancelGame := context.WithCancel(parent)

	// Wire game state checkpoint
	if gm.gameStateRepo != nil {
		game.SetOnSave(func(ctx context.Context, data mafia.CheckpointData) error {
			return gm.gameStateRepo.Save(ctx, repository.SavedGameState{
				RoomID:     data.RoomID,
				Phase:      string(data.Phase),
				Round:      data.Round,
				Players:    repository.ToSavedPlayers(data.Players),
				NightKills: data.NightKills,
			})
		})
	}

	// Spawn AI agents
	gm.ai.SpawnAgents(gameCtx, room.ID, room.GetPlayers(), personas, preloadedHistories)

	startedAt := time.Now()

	go func() {
		// Heartbeat to maintain leader lock
		if gm.leaderLock != nil {
			go func() {
				ticker := time.NewTicker(10 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						gm.leaderLock.Heartbeat(gameCtx, room.ID)
					case <-gameCtx.Done():
						return
					}
				}
			}()
		}

		game.Start(gameCtx)
		room.SetStatus(entity.RoomStatusFinished)
		gm.logger.Info("game finished", zap.String("room_id", room.ID))

		cancelGame()

		// Release leader lock and delete checkpoint
		if gm.leaderLock != nil {
			gm.leaderLock.Release(context.Background(), room.ID)
		}
		if gm.gameStateRepo != nil {
			if err := gm.gameStateRepo.Delete(context.Background(), room.ID); err != nil {
				gm.logger.Error("failed to delete game state checkpoint",
					zap.String("room_id", room.ID), zap.Error(err))
			}
		}

		gm.mu.Lock()
		delete(gm.activeGames, room.ID)
		gm.mu.Unlock()
	}()

	// Forward game events to AI agents and WS clients
	go func() {
		for {
			select {
			case event := <-game.Subscribe():
				gm.ai.BroadcastEvent(room.ID, event)

				if event.Type == entity.EventPhaseChange {
					go gm.ai.SaveHistories(context.Background(), room.ID)
				}

				if event.Type == entity.EventGameOver {
					go gm.saveGameResult(room, event, startedAt)
				}

				if gm.GameEventFunc != nil {
					gm.GameEventFunc(room.ID, event)
				}

			case <-gameCtx.Done():
				// Drain remaining events after context cancellation
				for {
					select {
					case event := <-game.Subscribe():
						gm.ai.BroadcastEvent(room.ID, event)
						if event.Type == entity.EventGameOver {
							go gm.saveGameResult(room, event, startedAt)
						}
						if gm.GameEventFunc != nil {
							gm.GameEventFunc(room.ID, event)
						}
					default:
						return
					}
				}
			}
		}
	}()

	gm.activeGames[room.ID] = activeGame{game: game, cancel: cancelGame, ctx: gameCtx}
	return nil
}

func (gm *GameManager) saveGameResult(room *entity.Room, event entity.GameEvent, startedAt time.Time) {
	if gm.gameResultRepo == nil {
		return
	}
	winner, _ := event.Payload["winner"].(string)
	round, _ := event.Payload["round"].(int)
	durationSec := int(time.Since(startedAt).Seconds())

	players := room.GetPlayers()
	resultPlayers := make([]repository.GameResultPlayer, 0, len(players))
	for _, p := range players {
		resultPlayers = append(resultPlayers, repository.GameResultPlayer{
			PlayerID:   p.ID,
			PlayerName: p.Name,
			Role:       string(p.Role),
			IsAI:       p.IsAI,
			Survived:   p.IsAlive,
		})
	}

	if err := gm.gameResultRepo.Save(context.Background(), repository.GameResult{
		RoomID:      room.ID,
		WinnerTeam:  winner,
		RoundCount:  round,
		DurationSec: durationSec,
		Players:     resultPlayers,
	}); err != nil {
		gm.logger.Error("failed to save game result",
			zap.String("room_id", room.ID), zap.Error(err))
	}
}

// GetSnapshot returns a point-in-time snapshot of the active game state for roomID.
// Returns nil if no game is currently active.
func (gm *GameManager) GetSnapshot(roomID string) *dto.GameSnapshot {
	gm.mu.Lock()
	ag, ok := gm.activeGames[roomID]
	gm.mu.Unlock()
	if !ok {
		return nil
	}
	state := ag.game.State()
	votes := state.Votes
	if votes == nil {
		votes = make(map[string]string)
	}
	return &dto.GameSnapshot{
		Phase:             string(state.Phase),
		Round:             state.Round,
		TimerRemainingSec: state.TimerRemainingSeconds,
		AlivePlayerIDs:    state.AlivePlayers,
		Votes:             votes,
	}
}

func (gm *GameManager) TryRecover(ctx context.Context, roomID string) {
	gm.mu.Lock()
	_, active := gm.activeGames[roomID]
	gm.mu.Unlock()
	if active {
		return // already running locally
	}
	if gm.leaderLock == nil || gm.leaderLock.HasLeader(ctx, roomID) {
		return // another instance leads or no redis
	}

	// No leader — try to recover
	room, err := gm.roomSvc.GetByID(roomID)
	if err != nil {
		return
	}
	if room.GetStatus() != entity.RoomStatusPlaying {
		return
	}

	var preloaded map[string][]anthropic.MessageParam
	if gm.aiHistoryRepo != nil {
		preloaded, _ = gm.aiHistoryRepo.GetByRoom(ctx, roomID)
	}

	if err := gm.RecoverGame(ctx, room, preloaded); err != nil {
		gm.logger.Error("TryRecover failed", zap.String("room_id", roomID), zap.Error(err))
	} else {
		gm.logger.Info("game recovered via WS trigger", zap.String("room_id", roomID))
	}
}

// DispatchAction routes a game action directly to the active game instance.
// Used by AI agent callbacks to submit votes, kills, and investigations.
func (gm *GameManager) DispatchAction(roomID, playerID string, action entity.Action) error {
	gm.mu.Lock()
	ag, ok := gm.activeGames[roomID]
	gm.mu.Unlock()
	if !ok {
		return nil
	}
	return ag.game.HandleAction(playerID, action)
}

func (gm *GameManager) NotifyEvent(roomID string, event entity.GameEvent) {
	gm.mu.Lock()
	ag, ok := gm.activeGames[roomID]
	gm.mu.Unlock()
	if !ok {
		return
	}

	raw, ok := event.Payload["raw"]
	if !ok {
		return
	}
	req, ok := raw.(dto.ActionRequest)
	if !ok {
		return
	}

	var action entity.Action
	switch req.Type {
	case "chat":
		if req.Chat == nil {
			return
		}
		action = entity.Action{
			Type: "chat",
			Payload: map[string]any{
				"message":    req.Chat.Message,
				"mafia_only": req.Chat.MafiaOnly,
			},
		}
	case "vote":
		if req.Vote == nil {
			return
		}
		action = entity.Action{
			Type:    "vote",
			Payload: map[string]any{"target_id": req.Vote.TargetID},
		}
	case "kill", "investigate":
		if req.Night == nil {
			return
		}
		action = entity.Action{
			Type:    req.Night.ActionType,
			Payload: map[string]any{"target_id": req.Night.TargetID},
		}
	default:
		return
	}

	if err := ag.game.HandleAction(event.PlayerID, action); err != nil {
		gm.logger.Error("NotifyEvent: HandleAction failed",
			zap.String("room_id", roomID),
			zap.String("player_id", event.PlayerID),
			zap.String("action", req.Type),
			zap.Error(err))
	}
}

// StopGame cancels the active game context for roomID, stopping the game engine.
// Used when all human players have left (spec #3).
func (gm *GameManager) StopGame(roomID string) {
	gm.mu.Lock()
	ag, ok := gm.activeGames[roomID]
	if ok {
		ag.cancel()
		delete(gm.activeGames, roomID)
	}
	gm.mu.Unlock()
}

// ReplaceWithAI spawns a new AI agent to take over a human player's slot.
// Called when a human player disconnects during an active game (spec #7).
func (gm *GameManager) ReplaceWithAI(roomID, playerID string, role entity.Role) error {
	gm.mu.Lock()
	ag, ok := gm.activeGames[roomID]
	gm.mu.Unlock()
	if !ok {
		return nil
	}

	personas := gm.personaPool.Assign(1)
	gm.ai.AddAgent(ag.ctx, roomID, playerID, role, personas[0])
	return nil
}
