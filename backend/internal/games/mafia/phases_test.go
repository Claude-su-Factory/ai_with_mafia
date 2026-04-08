package mafia

import (
	"testing"

	"go.uber.org/zap"

	"ai-playground/internal/domain/entity"
)

// --- helpers ---

// newTestPlayers returns a standard 6-player set: 2 mafia, 1 police, 3 citizens.
func newTestPlayers() []*entity.Player {
	m1 := entity.NewPlayer("m1", "M1", false)
	m1.Role = entity.RoleMafia
	m2 := entity.NewPlayer("m2", "M2", false)
	m2.Role = entity.RoleMafia
	p1 := entity.NewPlayer("p1", "P1", false)
	p1.Role = entity.RolePolice
	c1 := entity.NewPlayer("c1", "C1", false)
	c1.Role = entity.RoleCitizen
	c2 := entity.NewPlayer("c2", "C2", false)
	c2.Role = entity.RoleCitizen
	c3 := entity.NewPlayer("c3", "C3", false)
	c3.Role = entity.RoleCitizen
	return []*entity.Player{m1, m2, p1, c1, c2, c3}
}

func newTestPM(players []*entity.Player) (*PhaseManager, chan entity.GameEvent) {
	state := NewGameState(players)
	eventCh := make(chan entity.GameEvent, 32)
	pm := NewPhaseManager(state, eventCh, Timers{}, zap.NewNop(), "room-test")
	return pm, eventCh
}

func drainEvents(ch chan entity.GameEvent) []entity.GameEvent {
	var events []entity.GameEvent
	for {
		select {
		case e := <-ch:
			events = append(events, e)
		default:
			return events
		}
	}
}

func playerByID(players []*entity.Player, id string) *entity.Player {
	for _, p := range players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// --- processVotes ---

func TestProcessVotes_SingleWinner(t *testing.T) {
	players := newTestPlayers()
	pm, eventCh := newTestPM(players)

	// c1 gets 3 votes — clear winner
	pm.state.Votes = map[string]string{
		"m1": "c1",
		"m2": "c1",
		"p1": "c1",
		"c2": "m1",
		"c3": "m2",
	}

	pm.processVotes()

	if playerByID(players, "c1").IsAlive {
		t.Error("c1 should be eliminated after receiving plurality of votes")
	}
	// All others still alive
	for _, id := range []string{"m1", "m2", "p1", "c2", "c3"} {
		if !playerByID(players, id).IsAlive {
			t.Errorf("player %s should still be alive", id)
		}
	}

	events := drainEvents(eventCh)
	found := false
	for _, e := range events {
		if e.Type == entity.EventKill {
			if pid, _ := e.Payload["player_id"].(string); pid == "c1" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected EventKill for c1, none received")
	}
}

func TestProcessVotes_Tie(t *testing.T) {
	players := newTestPlayers()
	pm, eventCh := newTestPM(players)

	// c1 and c2 each get 2 votes — tie, no execution
	pm.state.Votes = map[string]string{
		"m1": "c1",
		"m2": "c1",
		"p1": "c2",
		"c3": "c2",
	}

	pm.processVotes()

	for _, p := range players {
		if !p.IsAlive {
			t.Errorf("no one should die on a tie, but %s is dead", p.ID)
		}
	}

	events := drainEvents(eventCh)
	found := false
	for _, e := range events {
		if e.Type == entity.EventVote {
			if r, _ := e.Payload["result"].(string); r == "tie" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected EventVote with result='tie', none received")
	}
}

func TestProcessVotes_NoVotes(t *testing.T) {
	players := newTestPlayers()
	pm, eventCh := newTestPM(players)
	// Votes map is empty by default from NewGameState

	pm.processVotes()

	for _, p := range players {
		if !p.IsAlive {
			t.Errorf("expected no kills with no votes, but %s is dead", p.ID)
		}
	}
	if events := drainEvents(eventCh); len(events) != 0 {
		t.Errorf("expected no events with empty votes, got %d", len(events))
	}
}

// --- CheckWin ---

func TestCheckWin_CitizenWin_AllMafiaEliminated(t *testing.T) {
	players := newTestPlayers()
	pm, _ := newTestPM(players)

	playerByID(players, "m1").IsAlive = false
	playerByID(players, "m2").IsAlive = false

	if got := pm.CheckWin(); got != "citizen" {
		t.Errorf("expected 'citizen', got %q", got)
	}
}

func TestCheckWin_MafiaWin_EqualCount(t *testing.T) {
	players := newTestPlayers()
	pm, _ := newTestPM(players)

	// Kill 3 citizens/police → 2 mafia alive, 1 non-mafia alive → 2 >= 1 → mafia
	killed := 0
	for _, p := range players {
		if p.Role != entity.RoleMafia && killed < 3 {
			p.IsAlive = false
			killed++
		}
	}

	if got := pm.CheckWin(); got != "mafia" {
		t.Errorf("expected 'mafia', got %q", got)
	}
}

func TestCheckWin_GameContinues(t *testing.T) {
	players := newTestPlayers()
	pm, _ := newTestPM(players)
	// All 6 alive: 2 mafia vs 4 non-mafia → 2 < 4 → game continues

	if got := pm.CheckWin(); got != "" {
		t.Errorf("expected game-continues (\"\"), got %q", got)
	}
}

// --- processMafiaKill ---

func TestProcessMafiaKill_Consensus(t *testing.T) {
	players := newTestPlayers()
	pm, eventCh := newTestPM(players)

	// Both alive mafia agree on c1
	pm.state.NightKills = map[string]string{
		"m1": "c1",
		"m2": "c1",
	}

	pm.processMafiaKill()

	if playerByID(players, "c1").IsAlive {
		t.Error("c1 should be killed by unanimous mafia consensus")
	}
	for _, id := range []string{"m1", "m2", "p1", "c2", "c3"} {
		if !playerByID(players, id).IsAlive {
			t.Errorf("player %s should still be alive", id)
		}
	}

	events := drainEvents(eventCh)
	found := false
	for _, e := range events {
		if e.Type == entity.EventKill {
			if pid, _ := e.Payload["player_id"].(string); pid == "c1" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected EventKill for c1 on mafia consensus")
	}
}

func TestProcessMafiaKill_NoConsensus(t *testing.T) {
	players := newTestPlayers()
	pm, eventCh := newTestPM(players)

	// Mafia disagree — no kill
	pm.state.NightKills = map[string]string{
		"m1": "c1",
		"m2": "c2",
	}

	pm.processMafiaKill()

	for _, p := range players {
		if !p.IsAlive {
			t.Errorf("expected no kills with no consensus, but %s is dead", p.ID)
		}
	}
	if events := drainEvents(eventCh); len(events) != 0 {
		t.Errorf("expected no events on no consensus, got %d", len(events))
	}
}

func TestProcessMafiaKill_SingleMafiaAutoConsensus(t *testing.T) {
	players := newTestPlayers()
	pm, eventCh := newTestPM(players)

	// Kill m2 so only m1 remains — single mafia vote counts as consensus
	playerByID(players, "m2").IsAlive = false
	pm.state.NightKills = map[string]string{
		"m1": "c3",
	}

	pm.processMafiaKill()

	if playerByID(players, "c3").IsAlive {
		t.Error("c3 should be killed by the sole surviving mafia")
	}

	events := drainEvents(eventCh)
	found := false
	for _, e := range events {
		if e.Type == entity.EventKill {
			if pid, _ := e.Payload["player_id"].(string); pid == "c3" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected EventKill for c3")
	}
}

func TestProcessMafiaKill_NoKillsSubmitted(t *testing.T) {
	players := newTestPlayers()
	pm, eventCh := newTestPM(players)
	// NightKills map is empty

	pm.processMafiaKill()

	for _, p := range players {
		if !p.IsAlive {
			t.Errorf("expected no kills with no submissions, but %s is dead", p.ID)
		}
	}
	if events := drainEvents(eventCh); len(events) != 0 {
		t.Errorf("expected no events, got %d", len(events))
	}
}

// --- RecordVote ---

func TestRecordVote_AlivePlayer(t *testing.T) {
	players := newTestPlayers()
	pm, _ := newTestPM(players)

	pm.RecordVote("c1", "m1")

	if pm.state.Votes["c1"] != "m1" {
		t.Errorf("expected vote 'c1'→'m1', got %q", pm.state.Votes["c1"])
	}
}

func TestRecordVote_DeadPlayerIgnored(t *testing.T) {
	players := newTestPlayers()
	pm, _ := newTestPM(players)

	playerByID(players, "c1").IsAlive = false
	pm.RecordVote("c1", "m1")

	if _, ok := pm.state.Votes["c1"]; ok {
		t.Error("dead player's vote should be silently ignored")
	}
}

// --- RecordMafiaKill ---

func TestRecordMafiaKill_AliveMafia(t *testing.T) {
	players := newTestPlayers()
	pm, _ := newTestPM(players)

	pm.RecordMafiaKill("m1", "c1")

	if pm.state.NightKills["m1"] != "c1" {
		t.Errorf("expected kill 'm1'→'c1', got %q", pm.state.NightKills["m1"])
	}
}

func TestRecordMafiaKill_DeadMafiaIgnored(t *testing.T) {
	players := newTestPlayers()
	pm, _ := newTestPM(players)

	playerByID(players, "m1").IsAlive = false
	pm.RecordMafiaKill("m1", "c1")

	if _, ok := pm.state.NightKills["m1"]; ok {
		t.Error("dead mafia player's kill should be silently ignored")
	}
}

func TestRecordMafiaKill_NonMafiaIgnored(t *testing.T) {
	players := newTestPlayers()
	pm, _ := newTestPM(players)

	pm.RecordMafiaKill("c1", "m1") // c1 is a citizen

	if _, ok := pm.state.NightKills["c1"]; ok {
		t.Error("non-mafia player's kill should be silently ignored")
	}
}
