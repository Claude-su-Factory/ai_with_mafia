package platform

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
)

// mockHub satisfies the GameHub interface without a real game engine.
type mockHub struct {
	startGameErr   error
	restartGameErr error
}

func (m *mockHub) StartGame(_ string) error   { return m.startGameErr }
func (m *mockHub) RestartGame(_ string) error { return m.restartGameErr }
func (m *mockHub) ForceRemove(_, _ string)    {}

// setupApp builds a Fiber app with real RoomService (in-memory) and mockHub.
// userRepo and sessionRepo are nil — auth-dependent endpoints return 401.
func setupApp(t *testing.T) (*fiber.App, *RoomService) {
	t.Helper()
	svc := NewRoomService(nil, zap.NewNop())
	h := NewHandler(svc, &mockHub{}, nil, nil, nil, "")
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	h.RegisterRoutes(app)
	return app, svc
}

// jsonBody builds a JSON request body from a string.
func jsonBody(s string) io.Reader { return strings.NewReader(s) }

// ─── Auth-gated endpoints → 401 without JWT ─────────────────────────────────

func TestCreateRoom_Unauthorized(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("POST", "/api/rooms",
		jsonBody(`{"name":"테스트방","visibility":"public","max_humans":6}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", resp.StatusCode)
	}
}

func TestJoinRoom_Unauthorized(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/join",
		jsonBody(`{"player_name":"참가자"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", resp.StatusCode)
	}
}

func TestStartGame_Unauthorized(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/start", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", resp.StatusCode)
	}
}

func TestRestartGame_Unauthorized(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/restart", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", resp.StatusCode)
	}
}

func TestMe_Unauthorized(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("GET", "/api/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", resp.StatusCode)
	}
}

// ─── GET /api/rooms (public, no auth required) ──────────────────────────────

func TestListRooms_Empty(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("GET", "/api/rooms", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var rooms []dto.RoomResponse
	json.NewDecoder(resp.Body).Decode(&rooms) //nolint
	if len(rooms) != 0 {
		t.Errorf("expected empty list, got %d rooms", len(rooms))
	}
}

func TestListRooms_ReturnsCreatedRoom(t *testing.T) {
	app, svc := setupApp(t)

	// Create room directly via service (auth-gated HTTP endpoint can't be used without JWT)
	svc.Create(dto.CreateRoomRequest{
		Name: "공개방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장") //nolint

	listReq := httptest.NewRequest("GET", "/api/rooms", nil)
	resp, err := app.Test(listReq)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	var rooms []dto.RoomResponse
	json.NewDecoder(resp.Body).Decode(&rooms) //nolint
	if len(rooms) != 1 {
		t.Errorf("expected 1 room, got %d", len(rooms))
	}
	if len(rooms) > 0 && rooms[0].Name != "공개방" {
		t.Errorf("expected room name '공개방', got %q", rooms[0].Name)
	}
}

func TestListRooms_AIPlayersExcluded(t *testing.T) {
	app, svc := setupApp(t)

	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 6, Visibility: "public",
	}, "host-1", "방장")
	room.AddPlayer(newTestAIPlayer("ai-0"))
	room.AddPlayer(newTestAIPlayer("ai-1"))

	req := httptest.NewRequest("GET", "/api/rooms", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	var rooms []dto.RoomResponse
	json.NewDecoder(resp.Body).Decode(&rooms) //nolint
	if len(rooms) == 0 {
		t.Fatal("expected 1 room")
	}
	for _, p := range rooms[0].Players {
		if p.IsAI {
			t.Errorf("AI player %q should be excluded from list response", p.ID)
		}
	}
	if len(rooms[0].Players) != 1 {
		t.Errorf("expected 1 human player, got %d", len(rooms[0].Players))
	}
}

// ─── GET /api/rooms/:id (public, no auth required) ──────────────────────────

func TestGetRoom_Success(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	req := httptest.NewRequest("GET", "/api/rooms/"+room.ID, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body dto.RoomResponse
	json.NewDecoder(resp.Body).Decode(&body) //nolint
	if body.ID != room.ID {
		t.Errorf("expected room id %q, got %q", room.ID, body.ID)
	}
}

func TestGetRoom_NotFound(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("GET", "/api/rooms/nonexistent-id", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ─── POST /api/rooms/:id/leave (no JWT, uses player_id in body) ─────────────

func TestLeaveRoom_MissingPlayerID(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/leave",
		jsonBody(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 for missing player_id, got %d", resp.StatusCode)
	}
}

func TestLeaveRoom_Success(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/leave",
		jsonBody(`{"player_id":"host-1"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestUpdateMe_Unauthorized(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("PUT", "/api/me",
		jsonBody(`{"display_name":"홍길동"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", resp.StatusCode)
	}
}

func TestMyStats_Unauthorized(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("GET", "/api/me/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", resp.StatusCode)
	}
}

func TestMyGames_Unauthorized(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("GET", "/api/me/games", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", resp.StatusCode)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newTestAIPlayer(id string) *entity.Player {
	return entity.NewPlayer(id, "AI봇", true)
}
