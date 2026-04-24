package ai

import (
	"context"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"go.uber.org/zap"

	"ai-playground/config"
	"ai-playground/internal/domain/entity"
	"ai-playground/internal/repository"
)

// MetricsSink is the repository.GameMetricsRepository surface used by Manager
// for emitting per-turn AI usage counters. Defined here so internal/ai does not
// depend on the concrete repository type at the type-parameter boundary — the
// real implementation lives in internal/repository, bridged via an adapter in
// main.go that performs the repository.AIUsage(u) value conversion.
type MetricsSink interface {
	AddAIUsage(ctx context.Context, gameID string, u AIUsage) error
}

type Manager struct {
	mu            sync.Mutex
	agents        map[string]map[string]*Agent // roomID -> playerID -> agent
	semaphore     chan struct{}
	pool          *PersonaPool
	cfg           *config.AIConfig
	client        *anthropic.Client
	logger        *zap.Logger
	aiHistoryRepo *repository.AIHistoryRepository
	metrics       MetricsSink

	// outCh receives outputs from all agents and routes them
	broadcast func(roomID string, playerID, playerName, message string, mafiaOnly bool)
	voteFunc  func(roomID, playerID, targetID string)
	nightFunc func(roomID, playerID, actionType, targetID string)
}

func NewManager(
	cfg *config.AIConfig,
	pool *PersonaPool,
	apiKey string,
	logger *zap.Logger,
	historyRepo *repository.AIHistoryRepository,
	metrics MetricsSink,
) *Manager {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Manager{
		agents:        make(map[string]map[string]*Agent),
		semaphore:     make(chan struct{}, cfg.MaxConcurrent),
		pool:          pool,
		cfg:           cfg,
		client:        &client,
		logger:        logger,
		aiHistoryRepo: historyRepo,
		metrics:       metrics,
	}
}

func (m *Manager) SetCallbacks(
	broadcast func(roomID string, playerID, playerName, message string, mafiaOnly bool),
	vote func(roomID, playerID, targetID string),
	night func(roomID, playerID, actionType, targetID string),
) {
	m.broadcast = broadcast
	m.voteFunc = vote
	m.nightFunc = night
}

// SaveHistories persists all agent histories for a room to DB.
func (m *Manager) SaveHistories(ctx context.Context, roomID string) {
	if m.aiHistoryRepo == nil {
		return
	}
	m.mu.Lock()
	agents := m.agents[roomID]
	m.mu.Unlock()

	for _, a := range agents {
		if err := m.aiHistoryRepo.Save(ctx, roomID, a.PlayerID, a.history); err != nil {
			m.logger.Error("failed to save ai history",
				zap.String("room_id", roomID),
				zap.String("player_id", a.PlayerID),
				zap.Error(err))
		}
	}
}

// SpawnAgents creates AI agents for a room based on player list.
// players with IsAI=true will get an agent; personas must be pre-assigned
// (same order as AI players appear in players slice).
// preloadedHistories may be nil (fresh start) or contain histories from a previous session (recovery).
// gameID is the unified identifier shared with game_results.id / game_metrics.game_id.
func (m *Manager) SpawnAgents(ctx context.Context, roomID, gameID string, players []*entity.Player, personas []Persona, preloadedHistories map[string][]anthropic.MessageParam) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 기존 에이전트 정리
	if prev, ok := m.agents[roomID]; ok {
		for _, a := range prev {
			_ = a // context cancel로 이미 종료됨
		}
	}

	aiPlayers := make([]*entity.Player, 0)
	for _, p := range players {
		if p.IsAI {
			aiPlayers = append(aiPlayers, p)
		}
	}

	// 마피아 공범 맵 구성
	mafiaIDs := make([]string, 0)
	for _, p := range players {
		if p.Role == entity.RoleMafia {
			mafiaIDs = append(mafiaIDs, p.ID)
		}
	}

	agents := make(map[string]*Agent, len(aiPlayers))

	for i, p := range aiPlayers {
		allies := make([]string, 0)
		if p.Role == entity.RoleMafia {
			for _, id := range mafiaIDs {
				if id != p.ID {
					allies = append(allies, id)
				}
			}
		}
		a := NewAgent(p.ID, personas[i], p.Role, allies, m.cfg, m.client, m.logger)
		if preloadedHistories != nil {
			if h, ok := preloadedHistories[p.ID]; ok {
				a.history = h
			}
		}
		// Wire per-turn usage hook into the metrics sink. gameID is the unified
		// id shared with game_results.id and game_metrics.game_id (T21). Closure
		// captures ctx + gameID so agent goroutines don't need to know about
		// metrics.
		if m.metrics != nil {
			sink := m.metrics
			gid := gameID
			a.onUsage = func(u AIUsage) {
				_ = sink.AddAIUsage(ctx, gid, u)
			}
		}
		agents[p.ID] = a

		// Use select+ctx so the goroutine exits when the game context is cancelled,
		// preventing a goroutine leak (outCh is never closed explicitly).
		go func(agent *Agent, rID string, ctx context.Context) {
			for {
				select {
				case out, ok := <-agent.Output():
					if !ok {
						return
					}
					select {
					case m.semaphore <- struct{}{}:
						m.handleOutput(rID, out)
						<-m.semaphore
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(a, roomID, ctx)

		go a.Run(ctx)
	}

	m.agents[roomID] = agents
}

func (m *Manager) handleOutput(roomID string, out AgentOutput) {
	switch out.ActionType {
	case "vote":
		if m.voteFunc != nil {
			m.voteFunc(roomID, out.PlayerID, out.TargetID)
		}
	case "kill":
		if m.nightFunc != nil {
			m.nightFunc(roomID, out.PlayerID, "kill", out.TargetID)
		}
	case "investigate":
		if m.nightFunc != nil {
			m.nightFunc(roomID, out.PlayerID, "investigate", out.TargetID)
		}
	default:
		// Empty ActionType = chat message
		if m.broadcast != nil {
			m.broadcast(roomID, out.PlayerID, out.PlayerName, out.Message, out.MafiaOnly)
		}
	}
}

// AddAgent creates and starts a single AI agent at runtime (e.g. mid-game AI backfill).
// gameID must match the active game's unified id so per-turn metrics are accumulated
// under the correct game_metrics row.
func (m *Manager) AddAgent(ctx context.Context, roomID, gameID string, playerID string, role entity.Role, persona Persona) {
	m.mu.Lock()

	// Collect mafia allies from existing agents in this room
	mafiaIDs := make([]string, 0)
	if agents, ok := m.agents[roomID]; ok {
		for pid, a := range agents {
			if a.Role == entity.RoleMafia {
				mafiaIDs = append(mafiaIDs, pid)
			}
		}
	}
	// If the new agent is mafia, it should also be in the allies list for existing agents
	if role == entity.RoleMafia {
		mafiaIDs = append(mafiaIDs, playerID)
	}

	// Build allies list for the new agent (exclude self)
	allies := make([]string, 0)
	if role == entity.RoleMafia {
		for _, id := range mafiaIDs {
			if id != playerID {
				allies = append(allies, id)
			}
		}
	}

	a := NewAgent(playerID, persona, role, allies, m.cfg, m.client, m.logger)

	// Wire per-turn usage hook (gameID is the unified id — see SpawnAgents).
	if m.metrics != nil {
		sink := m.metrics
		gid := gameID
		a.onUsage = func(u AIUsage) {
			_ = sink.AddAIUsage(ctx, gid, u)
		}
	}

	if m.agents[roomID] == nil {
		m.agents[roomID] = make(map[string]*Agent)
	}
	m.agents[roomID][playerID] = a

	m.mu.Unlock()

	// Start output consumer goroutine (same pattern as SpawnAgents)
	go func(agent *Agent, rID string, ctx context.Context) {
		for {
			select {
			case out, ok := <-agent.Output():
				if !ok {
					return
				}
				select {
				case m.semaphore <- struct{}{}:
					m.handleOutput(rID, out)
					<-m.semaphore
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}(a, roomID, ctx)

	go a.Run(ctx)
}

// BroadcastEvent sends a game event to all AI agents in a room.
func (m *Manager) BroadcastEvent(roomID string, event entity.GameEvent) {
	m.mu.Lock()
	agents := m.agents[roomID]
	m.mu.Unlock()

	for _, a := range agents {
		// 마피아 전용 이벤트는 마피아에게만
		if event.MafiaOnly && a.Role != entity.RoleMafia {
			continue
		}
		a.Send(event)
	}
}

// RemoveRoom cleans up agents for a room.
func (m *Manager) RemoveRoom(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, roomID)
}
