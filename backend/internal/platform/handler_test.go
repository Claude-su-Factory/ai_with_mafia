package platform

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
	"ai-playground/internal/repository"
)

// ─── mockUserStore — satisfies UserStore without a real DB ───────────────────

type mockUserStore struct {
	playerID    string
	displayName string
	updateErr   error
	getErr      error // if set, GetOrCreate returns this error (simulate DB outage)
}

func (m *mockUserStore) GetOrCreate(_ context.Context, _, _ string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	return m.playerID, nil
}
func (m *mockUserStore) GetDisplayName(_ context.Context, _ string) (string, error) {
	return m.displayName, nil
}
func (m *mockUserStore) UpdateDisplayName(_ context.Context, _, _ string) error {
	return m.updateErr
}

// ─── setupAppWithAuth — Fiber app with JWT key pair wired up ─────────────────

func setupAppWithAuth(t *testing.T) (*fiber.App, *RoomService, func(sub string) string) {
	t.Helper()
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	svc := NewRoomService(nil, zap.NewNop())
	userStore := &mockUserStore{playerID: "player-1", displayName: "테스터"}
	h := NewHandler(svc, &mockHub{}, userStore, nil, nil, &privKey.PublicKey)
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	h.RegisterRoutes(app)

	makeToken := func(sub string) string {
		claims := jwt.MapClaims{
			"sub": sub,
			"exp": time.Now().Add(time.Hour).Unix(),
			"user_metadata": map[string]any{"full_name": "테스터"},
		}
		tok, signErr := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(privKey)
		if signErr != nil {
			t.Fatalf("makeToken: %v", signErr)
		}
		return tok
	}

	return app, svc, makeToken
}

// bearerHeader returns an Authorization header value.
func bearerHeader(tok string) string { return "Bearer " + tok }

// Ensure repository types still satisfy the new interfaces at compile time.
var _ UserStore = (*repository.UserRepository)(nil)
var _ GameResultStore = (*repository.GameResultRepository)(nil)

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
	h := NewHandler(svc, &mockHub{}, nil, nil, nil, nil)
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

// ─── PUT /api/me validation ──────────────────────────────────────────────────

func TestUpdateMe_EmptyName(t *testing.T) {
	app, _, makeToken := setupAppWithAuth(t)
	tok := makeToken("user-1")

	req := httptest.NewRequest("PUT", "/api/me", jsonBody(`{"display_name":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", resp.StatusCode)
	}
}

func TestUpdateMe_TooLongName(t *testing.T) {
	app, _, makeToken := setupAppWithAuth(t)
	tok := makeToken("user-1")

	longName := strings.Repeat("가", 51) // 51 runes
	req := httptest.NewRequest("PUT", "/api/me",
		jsonBody(`{"display_name":"`+longName+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 for 51-rune name, got %d", resp.StatusCode)
	}
}

func TestUpdateMe_Valid(t *testing.T) {
	app, _, makeToken := setupAppWithAuth(t)
	tok := makeToken("user-1")

	req := httptest.NewRequest("PUT", "/api/me",
		jsonBody(`{"display_name":"새닉네임"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body) //nolint
	if body["display_name"] != "새닉네임" {
		t.Errorf("expected display_name '새닉네임', got %v", body["display_name"])
	}
}

// ─── GET /api/me/stats — nil gameResultRepo ──────────────────────────────────

func TestMyStats_NilGameRepo_ReturnsZeros(t *testing.T) {
	app, _, makeToken := setupAppWithAuth(t)
	tok := makeToken("user-1")

	req := httptest.NewRequest("GET", "/api/me/stats", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body) //nolint
	for _, field := range []string{"total_games", "wins", "losses"} {
		if v, _ := body[field].(float64); v != 0 {
			t.Errorf("expected %s=0, got %v", field, v)
		}
	}
	if v, _ := body["win_rate"].(float64); v != 0 {
		t.Errorf("expected win_rate=0, got %v", v)
	}
}

// ─── GET /api/me/games — nil gameResultRepo ──────────────────────────────────

func TestMyGames_NilGameRepo_ReturnsEmpty(t *testing.T) {
	app, _, makeToken := setupAppWithAuth(t)
	tok := makeToken("user-1")

	req := httptest.NewRequest("GET", "/api/me/games", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body []any
	json.NewDecoder(resp.Body).Decode(&body) //nolint
	if len(body) != 0 {
		t.Errorf("expected empty array, got %d items", len(body))
	}
}

// ─── POST /api/rooms/:id/start — authorization checks ────────────────────────

func TestStartGame_RoomNotFound(t *testing.T) {
	app, _, makeToken := setupAppWithAuth(t)
	tok := makeToken("some-user")

	req := httptest.NewRequest("POST", "/api/rooms/nonexistent-room/start", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404 for unknown room, got %d", resp.StatusCode)
	}
}

func TestStartGame_NonHost_Forbidden(t *testing.T) {
	app, svc, makeToken := setupAppWithAuth(t)
	// Room host is "host-1"; mockUserStore always resolves to playerID="player-1"
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	tok := makeToken("other-user")
	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/start", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403 for non-host, got %d", resp.StatusCode)
	}
}

func TestRestartGame_NonHost_Forbidden(t *testing.T) {
	app, svc, makeToken := setupAppWithAuth(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "방", MaxHumans: 4, Visibility: "public",
	}, "host-1", "방장")

	tok := makeToken("other-user")
	req := httptest.NewRequest("POST", "/api/rooms/"+room.ID+"/restart", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403 for non-host restart, got %d", resp.StatusCode)
	}
}

// ─── resolvePlayer nil userRepo → 500, not panic ─────────────────────────────

// setupAppNilUserRepo wires a real JWT key pair but leaves userRepo nil,
// so a valid token should return 500 (not a nil-pointer panic).
func setupAppNilUserRepo(t *testing.T) (*fiber.App, func(sub string) string) {
	t.Helper()
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	svc := NewRoomService(nil, zap.NewNop())
	h := NewHandler(svc, &mockHub{}, nil, nil, nil, &privKey.PublicKey)
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	h.RegisterRoutes(app)

	makeToken := func(sub string) string {
		claims := jwt.MapClaims{
			"sub": sub,
			"exp": time.Now().Add(time.Hour).Unix(),
			"user_metadata": map[string]any{"full_name": "테스터"},
		}
		tok, signErr := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(privKey)
		if signErr != nil {
			t.Fatalf("makeToken: %v", signErr)
		}
		return tok
	}
	return app, makeToken
}

func TestResolvePlayer_NilUserRepo_Returns500(t *testing.T) {
	app, makeToken := setupAppNilUserRepo(t)
	tok := makeToken("user-1")

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500 for nil userRepo with valid JWT, got %d", resp.StatusCode)
	}
}

// ─── Infrastructure failure (DB outage) → 500, not 401 ───────────────────────
//
// Reviewer-identified bug: plain DB errors from GetOrCreate used to fall through
// respondPlayerErr's fallback and surface as 401 Unauthorized — making a valid
// token look forged. Handler now wraps non-auth errors with fiber.ErrInternalServerError
// so errors.As classifies them as 500.

// setupAppWithUserStore wires a Handler around a caller-supplied UserStore so
// tests can inject failures (e.g. a mock that returns pgx-style errors).
func setupAppWithUserStore(t *testing.T, store UserStore) (*fiber.App, func(sub string) string) {
	t.Helper()
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	svc := NewRoomService(nil, zap.NewNop())
	h := NewHandler(svc, &mockHub{}, store, nil, nil, &privKey.PublicKey)
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	h.RegisterRoutes(app)

	makeToken := func(sub string) string {
		claims := jwt.MapClaims{
			"sub": sub,
			"exp": time.Now().Add(time.Hour).Unix(),
			"user_metadata": map[string]any{"full_name": "테스터"},
		}
		tok, signErr := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(privKey)
		if signErr != nil {
			t.Fatalf("makeToken: %v", signErr)
		}
		return tok
	}
	return app, makeToken
}

func TestMe_DBError_Returns500(t *testing.T) {
	store := &mockUserStore{getErr: errors.New("pgx: connection refused")}
	app, makeToken := setupAppWithUserStore(t, store)
	tok := makeToken("user-1")

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500 when userRepo.GetOrCreate fails with DB error, got %d", resp.StatusCode)
	}
}

func TestCreateRoom_DBError_Returns500(t *testing.T) {
	store := &mockUserStore{getErr: errors.New("pgx: connection refused")}
	app, makeToken := setupAppWithUserStore(t, store)
	tok := makeToken("user-1")

	req := httptest.NewRequest("POST", "/api/rooms",
		jsonBody(`{"name":"방","visibility":"public","max_humans":6}`))
	req.Header.Set("Authorization", bearerHeader(tok))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500 when resolvePlayerFull DB lookup fails, got %d", resp.StatusCode)
	}
}

func TestCreateRoom_NilUserRepo_Returns500(t *testing.T) {
	app, makeToken := setupAppNilUserRepo(t)
	tok := makeToken("user-1")

	req := httptest.NewRequest("POST", "/api/rooms",
		jsonBody(`{"name":"방","visibility":"public","max_humans":6}`))
	req.Header.Set("Authorization", bearerHeader(tok))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500 for nil userRepo via resolvePlayerFull, got %d", resp.StatusCode)
	}
}

// ─── POST /api/rooms/quick (Phase A §3-C) ────────────────────────────────────

func TestQuickMatch_NoPublicRoom_CreatesNew(t *testing.T) {
	app, _, makeToken := setupAppWithAuth(t)
	tok := makeToken("user-1")

	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		RoomID   string `json:"room_id"`
		PlayerID string `json:"player_id"`
		Created  bool   `json:"created"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Created {
		t.Error("created = false, want true")
	}
	if body.RoomID == "" {
		t.Error("room_id is empty")
	}
}

func TestQuickMatch_PublicRoomFull_CreatesNew(t *testing.T) {
	app, svc, makeToken := setupAppWithAuth(t)

	// Fill one public room to capacity 2
	full, _ := svc.Create(dto.CreateRoomRequest{
		Name: "가득", MaxHumans: 2, Visibility: "public",
	}, "host-1", "H1")
	svc.Join(full.ID, "player-x", "X")

	tok := makeToken("user-1")
	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var body struct {
		Created bool `json:"created"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !body.Created {
		t.Error("created = false, want true (existing room was full)")
	}
}

func TestQuickMatch_PublicRoomAvailable_Joins(t *testing.T) {
	app, svc, makeToken := setupAppWithAuth(t)

	target, _ := svc.Create(dto.CreateRoomRequest{
		Name: "합류대상", MaxHumans: 4, Visibility: "public",
	}, "host-1", "H1")

	tok := makeToken("user-1")
	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var body struct {
		RoomID  string `json:"room_id"`
		Created bool   `json:"created"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Created {
		t.Error("created = true, want false")
	}
	if body.RoomID != target.ID {
		t.Errorf("room_id = %s, want %s", body.RoomID, target.ID)
	}
}

func TestQuickMatch_IgnoresPrivateRoom(t *testing.T) {
	app, svc, makeToken := setupAppWithAuth(t)

	_, _ = svc.Create(dto.CreateRoomRequest{
		Name: "비밀", MaxHumans: 6, Visibility: "private",
	}, "host-1", "H1")

	tok := makeToken("user-1")
	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var body struct {
		Created bool `json:"created"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !body.Created {
		t.Error("created = false, want true (private rooms must be ignored)")
	}
}

func TestQuickMatch_Unauthorized(t *testing.T) {
	app, _, _ := setupAppWithAuth(t)

	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	// no Authorization header

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newTestAIPlayer(id string) *entity.Player {
	return entity.NewPlayer(id, "AI봇", true)
}
