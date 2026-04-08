package mafia

import (
	"context"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"ai-playground/internal/domain/entity"
)

// CheckpointData is a safe snapshot of game state passed to the onSave callback.
type CheckpointData struct {
	RoomID     string
	Phase      entity.Phase
	Round      int
	Players    []*entity.Player
	NightKills map[string]string
}

type PhaseManager struct {
	mu      sync.Mutex
	state   *GameState
	eventCh chan<- entity.GameEvent
	timers  Timers
	logger  *zap.Logger
	roomID  string
	onSave  func(ctx context.Context, data CheckpointData) error
}

type Timers struct {
	DayDiscussion time.Duration
	DayVote       time.Duration
	Night         time.Duration
}

type GameState struct {
	mu             sync.RWMutex
	Phase          entity.Phase
	Players        []*entity.Player
	Votes          map[string]string // voterID -> targetID
	NightKills     map[string]string // killerID -> targetID (마피아 투표)
	Investigated   map[string]string // policeID -> targetID (조사 결과)
	Round          int
	WinnerTeam     string
	phaseStartedAt time.Time
	phaseDuration  time.Duration
}

func NewGameState(players []*entity.Player) *GameState {
	return &GameState{
		Phase:        entity.PhaseDayDiscussion,
		Players:      players,
		Votes:        make(map[string]string),
		NightKills:   make(map[string]string),
		Investigated: make(map[string]string),
	}
}

func NewPhaseManager(state *GameState, eventCh chan<- entity.GameEvent, timers Timers, logger *zap.Logger, roomID string) *PhaseManager {
	return &PhaseManager{state: state, eventCh: eventCh, timers: timers, logger: logger, roomID: roomID}
}

// SetOnSave sets the checkpoint callback. Called after phase starts.
func (pm *PhaseManager) SetOnSave(fn func(ctx context.Context, data CheckpointData) error) {
	pm.onSave = fn
}

func (pm *PhaseManager) save(ctx context.Context) {
	if pm.onSave == nil {
		return
	}
	// Build a safe snapshot while holding RLock
	pm.state.mu.RLock()
	players := make([]*entity.Player, len(pm.state.Players))
	for i, p := range pm.state.Players {
		cp := *p
		players[i] = &cp
	}
	nightKills := make(map[string]string, len(pm.state.NightKills))
	for k, v := range pm.state.NightKills {
		nightKills[k] = v
	}
	data := CheckpointData{
		RoomID:     pm.roomID,
		Phase:      pm.state.Phase,
		Round:      pm.state.Round,
		Players:    players,
		NightKills: nightKills,
	}
	pm.state.mu.RUnlock()

	if err := pm.onSave(ctx, data); err != nil {
		pm.logger.Error("game state checkpoint failed", zap.Error(err))
	}
}

// RunDayDiscussion broadcasts phase start and waits for timer.
func (pm *PhaseManager) RunDayDiscussion(ctx context.Context) {
	pm.state.mu.Lock()
	pm.state.Phase = entity.PhaseDayDiscussion
	pm.state.Votes = make(map[string]string)
	pm.state.phaseStartedAt = time.Now()
	pm.state.phaseDuration = pm.timers.DayDiscussion
	pm.state.mu.Unlock()

	pm.save(ctx)

	pm.emit(entity.GameEvent{
		Type: entity.EventPhaseChange,
		Payload: map[string]any{
			"phase":         string(entity.PhaseDayDiscussion),
			"duration":      int(pm.timers.DayDiscussion.Seconds()),
			"alive_players": pm.aliveIDs(),
			"round":         pm.state.Round,
		},
	})

	select {
	case <-time.After(pm.timers.DayDiscussion):
	case <-ctx.Done():
		return
	}
}

// RunDayVote collects votes and processes result.
func (pm *PhaseManager) RunDayVote(ctx context.Context) {
	pm.state.mu.Lock()
	pm.state.Phase = entity.PhaseDayVote
	pm.state.phaseStartedAt = time.Now()
	pm.state.phaseDuration = pm.timers.DayVote
	pm.state.mu.Unlock()

	pm.save(ctx)

	pm.emit(entity.GameEvent{
		Type: entity.EventPhaseChange,
		Payload: map[string]any{
			"phase":         string(entity.PhaseDayVote),
			"duration":      int(pm.timers.DayVote.Seconds()),
			"alive_players": pm.aliveIDs(),
			"round":         pm.state.Round,
		},
	})

	select {
	case <-time.After(pm.timers.DayVote):
	case <-ctx.Done():
		return
	}

	pm.processVotes()
}

func (pm *PhaseManager) processVotes() {
	pm.state.mu.Lock()
	defer pm.state.mu.Unlock()

	tally := make(map[string]int)
	for _, targetID := range pm.state.Votes {
		tally[targetID]++
	}

	if len(tally) == 0 {
		return
	}

	maxVotes := 0
	for _, cnt := range tally {
		if cnt > maxVotes {
			maxVotes = cnt
		}
	}

	// 최다 득표자가 여럿이면 동표 → 처형 없음
	var topCandidates []string
	for id, cnt := range tally {
		if cnt == maxVotes {
			topCandidates = append(topCandidates, id)
		}
	}
	sort.Strings(topCandidates)

	if len(topCandidates) > 1 {
		pm.emit(entity.GameEvent{
			Type:    entity.EventVote,
			Payload: map[string]any{"result": "tie", "votes": pm.state.Votes},
		})
		return
	}

	// 처형
	targetID := topCandidates[0]
	for _, p := range pm.state.Players {
		if p.ID == targetID {
			p.IsAlive = false
			pm.emit(entity.GameEvent{
				Type: entity.EventKill,
				Payload: map[string]any{
					"player_id": targetID,
					"role":      string(p.Role),
					"reason":    "vote",
				},
			})
			break
		}
	}
}

// RunNight handles night phase: mafia kill + police investigate.
func (pm *PhaseManager) RunNight(ctx context.Context) {
	pm.state.mu.Lock()
	pm.state.Phase = entity.PhaseNight
	pm.state.NightKills = make(map[string]string)
	pm.state.phaseStartedAt = time.Now()
	pm.state.phaseDuration = pm.timers.Night
	pm.state.mu.Unlock()

	pm.save(ctx)

	// 마피아 채널 오픈
	pm.emit(entity.GameEvent{
		Type:      entity.EventPhaseChange,
		MafiaOnly: true,
		Payload: map[string]any{
			"phase":         string(entity.PhaseNight),
			"duration":      int(pm.timers.Night.Seconds()),
			"alive_players": pm.aliveIDs(),
			"message":       "밤이 되었습니다. 처치할 대상을 상의하세요.",
			"round":         pm.state.Round,
		},
	})
	// 시민에게는 밤 시작만 알림
	pm.emit(entity.GameEvent{
		Type: entity.EventPhaseChange,
		Payload: map[string]any{
			"phase":    string(entity.PhaseNight),
			"duration": int(pm.timers.Night.Seconds()),
			"round":    pm.state.Round,
		},
	})

	select {
	case <-time.After(pm.timers.Night):
	case <-ctx.Done():
		return
	}

	pm.processMafiaKill()
}

func (pm *PhaseManager) processMafiaKill() {
	pm.state.mu.Lock()
	defer pm.state.mu.Unlock()

	if len(pm.state.NightKills) == 0 {
		return
	}

	// 마피아 전원이 같은 대상을 선택했는지 확인
	mafiaCount := 0
	tally := make(map[string]int)
	for _, p := range pm.state.Players {
		if p.Role == entity.RoleMafia && p.IsAlive {
			mafiaCount++
		}
	}
	for _, targetID := range pm.state.NightKills {
		tally[targetID]++
	}

	// 합의된 대상 찾기
	for targetID, cnt := range tally {
		if cnt == mafiaCount {
			for _, p := range pm.state.Players {
				if p.ID == targetID {
					p.IsAlive = false
					pm.emit(entity.GameEvent{
						Type: entity.EventKill,
						Payload: map[string]any{
							"player_id": targetID,
							"reason":    "mafia_kill",
						},
					})
					return
				}
			}
		}
	}
	// 합의 없음 → 아무도 안 죽음
}

// RecordVote records a day vote. Dead players are silently ignored.
func (pm *PhaseManager) RecordVote(voterID, targetID string) {
	pm.state.mu.Lock()
	defer pm.state.mu.Unlock()

	// Only alive players may vote
	var voter *entity.Player
	for _, p := range pm.state.Players {
		if p.ID == voterID {
			voter = p
			break
		}
	}
	if voter == nil || !voter.IsAlive {
		return
	}

	pm.state.Votes[voterID] = targetID
	pm.emit(entity.GameEvent{
		Type: entity.EventVote,
		Payload: map[string]any{
			"voter_id":  voterID,
			"target_id": targetID,
		},
	})
}

// RecordMafiaKill records a mafia night kill vote. Only alive mafia players are accepted.
func (pm *PhaseManager) RecordMafiaKill(killerID, targetID string) {
	pm.state.mu.Lock()
	defer pm.state.mu.Unlock()

	// Only alive mafia players may submit kills
	var killer *entity.Player
	for _, p := range pm.state.Players {
		if p.ID == killerID {
			killer = p
			break
		}
	}
	if killer == nil || killer.Role != entity.RoleMafia || !killer.IsAlive {
		return
	}

	pm.state.NightKills[killerID] = targetID
}

// RecordInvestigation records police investigation and returns result.
func (pm *PhaseManager) RecordInvestigation(policeID, targetID string) {
	pm.state.mu.Lock()
	isMafia := false
	for _, p := range pm.state.Players {
		if p.ID == targetID {
			isMafia = p.Role == entity.RoleMafia
			break
		}
	}
	pm.state.Investigated[policeID] = targetID
	pm.state.mu.Unlock()

	// 조사 결과는 경찰에게만
	pm.emit(entity.GameEvent{
		Type:     entity.EventNightAction,
		PlayerID: policeID,
		Payload: map[string]any{
			"type":      "investigation_result",
			"target_id": targetID,
			"is_mafia":  isMafia,
		},
	})
}

// CheckWin returns "" if game continues, "mafia" or "citizen" if won.
func (pm *PhaseManager) CheckWin() string {
	pm.state.mu.RLock()
	defer pm.state.mu.RUnlock()

	aliveMafia := 0
	aliveOthers := 0
	for _, p := range pm.state.Players {
		if !p.IsAlive {
			continue
		}
		if p.Role == entity.RoleMafia {
			aliveMafia++
		} else {
			aliveOthers++
		}
	}

	if aliveMafia == 0 {
		return "citizen"
	}
	if aliveMafia >= aliveOthers {
		return "mafia"
	}
	return ""
}

func (pm *PhaseManager) State() entity.GameState {
	pm.state.mu.RLock()
	defer pm.state.mu.RUnlock()

	aliveIDs := make([]string, 0)
	for _, p := range pm.state.Players {
		if p.IsAlive {
			aliveIDs = append(aliveIDs, p.ID)
		}
	}
	votes := make(map[string]string, len(pm.state.Votes))
	for k, v := range pm.state.Votes {
		votes[k] = v
	}

	var timerRemaining int
	if pm.state.phaseDuration > 0 {
		remaining := pm.state.phaseDuration - time.Since(pm.state.phaseStartedAt)
		if remaining < 0 {
			remaining = 0
		}
		timerRemaining = int(remaining.Seconds())
	}

	return entity.GameState{
		Phase:                 pm.state.Phase,
		AlivePlayers:          aliveIDs,
		Votes:                 votes,
		Round:                 pm.state.Round,
		WinnerTeam:            pm.state.WinnerTeam,
		TimerRemainingSeconds: timerRemaining,
	}
}

func (pm *PhaseManager) aliveIDs() []string {
	var ids []string
	for _, p := range pm.state.Players {
		if p.IsAlive {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

func (pm *PhaseManager) emit(event entity.GameEvent) {
	select {
	case pm.eventCh <- event:
	default:
		pm.logger.Warn("event channel full, dropping event", zap.String("type", string(event.Type)))
	}
}
