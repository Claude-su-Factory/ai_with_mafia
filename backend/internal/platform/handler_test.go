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

// setupApp builds a Fiber app with real RoomService (in-memory) and mockHub.
func setupApp(t *testing.T) (*fiber.App, *RoomService) {
	t.Helper()
	svc := NewRoomService(nil, zap.NewNop())
	h := NewHandler(svc, &mockHub{})
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

// ─── POST /api/rooms ─────────────────────────────────────────────────────────

func TestCreateRoom_Success(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("POST", "/api/rooms",
		jsonBody(`{"name":"테스트방","visibility":"public","max_humans":6}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Player-Name", "방장")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var body dto.JoinRoomResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.PlayerID == "" {
		t.Error("response should include player_id")
	}
	if body.ID == "" {
		t.Error("response should include room id")
	}
	if body.Name != "테스트방" {
		t.Errorf("expected name '테스트방', got %q", body.Name)
	}
	if body.Status != "waiting" {
		t.Errorf("expected status 'waiting', got %q", body.Status)
	}
	// 방장 본인이 players에 포함돼야 함 (AI 제외이므로 1명)
	if len(body.Players) != 1 {
		t.Errorf("expected 1 player in response, got %d", len(body.Players))
	}
}

func TestCreateRoom_DefaultHostName(t *testing.T) {
	app, _ := setupApp(t)

	// X-Player-Name 헤더 없이 요청
	req := httptest.NewRequest("POST", "/api/rooms",
		jsonBody(`{"name":"방","visibility":"public","max_humans":4}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	var body dto.JoinRoomResponse
	json.NewDecoder(resp.Body).Decode(&body) //nolint
	if body.Players[0].Name != "방장" {
		t.Errorf("expected default host name '방장', got %q", body.Players[0].Name)
	}
}

func TestCreateRoom_InvalidMaxHumans(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("POST", "/api/rooms",
		jsonBody(`{"name":"방","visibility":"public","max_humans":10}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Player-Name", "방장")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 for invalid max_humans, got %d", resp.StatusCode)
	}
}

func TestCreateRoom_PrivateRoom_HasJoinCode(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("POST", "/api/rooms",
		jsonBody(`{"name":"비공개방","visibility":"private","max_humans":4}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Player-Name", "방장")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	var body dto.JoinRoomResponse
	json.NewDecoder(resp.Body).Decode(&body) //nolint
	if body.JoinCode == "" {
		t.Error("private room response should include join_code")
	}
}

// ─── GET /api/rooms ──────────────────────────────────────────────────────────

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
	app, _ := setupApp(t)

	// 방 생성
	createReq := httptest.NewRequest("POST", "/api/rooms",
		jsonBody(`{"name":"공개방","visibility":"public","max_humans":4}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-Player-Name", "방장")
	app.Test(createReq) //nolint

	// 목록 조회
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
	if rooms[0].Name != "공개방" {
		t.Errorf("expected room name '공개방', got %q", rooms[0].Name)
	}
}

func TestListRooms_AIPlayersExcluded(t *testing.T) {
	app, svc := setupApp(t)

	// 방 생성 후 AI 플레이어 직접 추가
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
	// players에 AI가 없어야 함
	for _, p := range rooms[0].Players {
		if p.IsAI {
			t.Errorf("AI player %q should be excluded from list response", p.ID)
		}
	}
	// 인간 1명만 표시
	if len(rooms[0].Players) != 1 {
		t.Errorf("expected 1 human player, got %d", len(rooms[0].Players))
	}
}

// ─── GET /api/rooms/:id ──────────────────────────────────────────────────────

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

// ─── POST /api/rooms/:id/join ─────────────────────────────────────────────────

func TestJoinRoom_Success(t *testing.T) {
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
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body dto.JoinRoomResponse
	json.NewDecoder(resp.Body).Decode(&body) //nolint
	if body.PlayerID == "" {
		t.Error("join response should include player_id")
	}
	if len(body.Players) != 2 {
		t.Errorf("expected 2 players after join, got %d", len(body.Players))
	}
}

func TestJoinRoom_RoomFull(t *testing.T) {
	app, svc := setupApp(t)
	// MaxHumans=1 방 생성 (방장 1명 = 꽉 참)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 1, Visibility: "public",
	}, "host-1", "방장")

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/join",
		jsonBody(`{"player_name":"참가자"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Errorf("expected 409 Conflict for full room, got %d", resp.StatusCode)
	}
}

func TestJoinRoom_NotFound(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("POST", "/api/rooms/nonexistent/join",
		jsonBody(`{"player_name":"참가자"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ─── POST /api/rooms/:id/start ───────────────────────────────────────────────

func TestStartGame_Success(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/start", nil)
	req.Header.Set("X-Player-ID", "host-1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestStartGame_Forbidden_NonHost(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")
	svc.Join(room.ID, "player-2", "참가자") //nolint

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/start", nil)
	req.Header.Set("X-Player-ID", "player-2") // 방장 아닌 사람

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403 Forbidden for non-host start, got %d", resp.StatusCode)
	}
}

func TestStartGame_NotFound(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("POST", "/api/rooms/nonexistent/start", nil)
	req.Header.Set("X-Player-ID", "host-1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestStartGame_AlreadyStarted(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")
	room.SetStatus("playing")

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/start", nil)
	req.Header.Set("X-Player-ID", "host-1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Errorf("expected 409 Conflict for already-started game, got %d", resp.StatusCode)
	}
}

// ─── POST /api/rooms/:id/restart ─────────────────────────────────────────────

func TestRestartGame_Success(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/restart", nil)
	req.Header.Set("X-Player-ID", "host-1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRestartGame_Forbidden_NonHost(t *testing.T) {
	app, svc := setupApp(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")
	svc.Join(room.ID, "player-2", "참가자") //nolint

	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/restart", nil)
	req.Header.Set("X-Player-ID", "player-2")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newTestAIPlayer(id string) *entity.Player {
	return entity.NewPlayer(id, "AI봇", true)
}
