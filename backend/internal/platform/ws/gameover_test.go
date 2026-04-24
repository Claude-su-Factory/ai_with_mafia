package ws

import (
	"encoding/json"
	"testing"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
)

// D1: "all_humans_left" path of game_over previously sent only {reason}, which
// crashed the frontend's GameOverResult expectation. The helper tested here
// provides a full-shape payload so the frontend can route & render safely.

func TestBuildAbortedGameOverPayload_HasAllRequiredFields(t *testing.T) {
	payload := buildAbortedGameOverPayload()

	required := []string{"winner", "round", "duration_sec", "players", "reason"}
	for _, key := range required {
		if _, ok := payload[key]; !ok {
			t.Errorf("payload missing required key %q", key)
		}
	}
}

func TestBuildAbortedGameOverPayload_WinnerIsAborted(t *testing.T) {
	payload := buildAbortedGameOverPayload()

	winner, ok := payload["winner"].(string)
	if !ok {
		t.Fatalf("winner is not a string: %T", payload["winner"])
	}
	if winner != "aborted" {
		t.Errorf("winner = %q, want %q", winner, "aborted")
	}
}

func TestBuildAbortedGameOverPayload_PlayersIsEmptySlice(t *testing.T) {
	payload := buildAbortedGameOverPayload()

	// Frontend GameOverResult.players is a non-nullable array.
	// The payload must be an empty slice (not nil) so JSON marshals as [].
	players, ok := payload["players"].([]map[string]any)
	if !ok {
		t.Fatalf("players type = %T, want []map[string]any", payload["players"])
	}
	if len(players) != 0 {
		t.Errorf("players len = %d, want 0", len(players))
	}

	// Verify it marshals to "[]" not "null"
	b, err := json.Marshal(players)
	if err != nil {
		t.Fatalf("marshal players: %v", err)
	}
	if string(b) != "[]" {
		t.Errorf("players JSON = %s, want []", string(b))
	}
}

func TestBuildAbortedGameOverPayload_ReasonIsAllHumansLeft(t *testing.T) {
	payload := buildAbortedGameOverPayload()

	reason, ok := payload["reason"].(string)
	if !ok {
		t.Fatalf("reason is not a string: %T", payload["reason"])
	}
	if reason != "all_humans_left" {
		t.Errorf("reason = %q, want %q", reason, "all_humans_left")
	}
}

func TestBuildAbortedGameOverPayload_MarshalsAsExpectedShape(t *testing.T) {
	// Integration-style: verify full JSON shape matches frontend GameOverResult contract.
	event := dto.GameEventDTO{
		Type:    string(entity.EventGameOver),
		Payload: buildAbortedGameOverPayload(),
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var parsed struct {
		Type    string `json:"type"`
		Payload struct {
			Winner      string           `json:"winner"`
			Round       int              `json:"round"`
			DurationSec int              `json:"duration_sec"`
			Players     []map[string]any `json:"players"`
			Reason      string           `json:"reason"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Type != "game_over" {
		t.Errorf("type = %q, want game_over", parsed.Type)
	}
	if parsed.Payload.Winner != "aborted" {
		t.Errorf("winner = %q, want aborted", parsed.Payload.Winner)
	}
	if parsed.Payload.Reason != "all_humans_left" {
		t.Errorf("reason = %q, want all_humans_left", parsed.Payload.Reason)
	}
	if parsed.Payload.Players == nil {
		t.Error("players is null, want []")
	}
}
