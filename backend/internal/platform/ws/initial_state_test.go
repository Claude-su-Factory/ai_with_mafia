package ws

import (
	"testing"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
)

// D2: WS initial_state previously hand-rolled a room payload that omitted
// max_humans, diverging from HTTP RoomResponse. The helper tested here is the
// single source of truth for "room as seen over WS" and must stay aligned
// with RoomResponse (plus the JoinCode visibility policy).

func TestBuildInitialStateRoomPayload_IncludesMaxHumans(t *testing.T) {
	room := newTestRoomForPayload(t, entity.VisibilityPublic, 6)

	payload := buildInitialStateRoomPayload(room)

	mh, ok := payload["max_humans"].(int)
	if !ok {
		t.Fatalf("max_humans type = %T, want int", payload["max_humans"])
	}
	if mh != 6 {
		t.Errorf("max_humans = %d, want 6", mh)
	}
}

func TestBuildInitialStateRoomPayload_AllRequiredFieldsPresent(t *testing.T) {
	room := newTestRoomForPayload(t, entity.VisibilityPublic, 4)

	payload := buildInitialStateRoomPayload(room)

	// All fields the frontend Room type declares must be present.
	required := []string{"id", "name", "status", "host_id", "visibility", "join_code", "max_humans", "players"}
	for _, key := range required {
		if _, ok := payload[key]; !ok {
			t.Errorf("payload missing key %q", key)
		}
	}
}

func TestBuildInitialStateRoomPayload_JoinCodeEmptyForPublic(t *testing.T) {
	// Public rooms get no meaningful join code — payload carries empty string
	// so the key is always present (frontend contract).
	room := newTestRoomForPayload(t, entity.VisibilityPublic, 4)

	payload := buildInitialStateRoomPayload(room)

	jc, ok := payload["join_code"].(string)
	if !ok {
		t.Fatalf("join_code type = %T, want string", payload["join_code"])
	}
	if jc != "" {
		t.Errorf("join_code = %q for public room, want empty", jc)
	}
}

func TestBuildInitialStateRoomPayload_JoinCodePopulatedForPrivate(t *testing.T) {
	room := newTestRoomForPayload(t, entity.VisibilityPrivate, 4)
	room.JoinCode = "ABCDEF"

	payload := buildInitialStateRoomPayload(room)

	jc, _ := payload["join_code"].(string)
	if jc != "ABCDEF" {
		t.Errorf("join_code = %q, want ABCDEF", jc)
	}
}

func TestBuildInitialStateRoomPayload_ExcludesAIPlayers(t *testing.T) {
	room := newTestRoomForPayload(t, entity.VisibilityPublic, 4)
	room.AddPlayer(entity.NewPlayer("ai-1", "AI봇", true))
	room.AddPlayer(entity.NewPlayer("human-1", "사람", false))

	payload := buildInitialStateRoomPayload(room)

	players, ok := payload["players"].([]dto.PlayerDTO)
	if !ok {
		t.Fatalf("players type = %T, want []dto.PlayerDTO", payload["players"])
	}
	for _, p := range players {
		if p.IsAI {
			t.Errorf("AI player %q should have been excluded", p.ID)
		}
	}
}

// newTestRoomForPayload constructs a minimal Room value suitable for testing
// payload builders. Kept intentionally tight — the platform.RoomService
// factory covers end-to-end coverage elsewhere.
func newTestRoomForPayload(t *testing.T, vis entity.Visibility, maxHumans int) *entity.Room {
	t.Helper()
	return &entity.Room{
		ID:         "room-1",
		Name:       "테스트방",
		HostID:     "host-1",
		MaxHumans:  maxHumans,
		Visibility: vis,
	}
}
