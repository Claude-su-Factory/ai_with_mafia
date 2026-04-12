package platform

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
	"ai-playground/internal/repository"
)

type Handler struct {
	rooms       *RoomService
	gameHub     GameHub
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
	jwtSecret   string
}

// GameHub is implemented by ws.Hub; defined here to avoid circular imports.
type GameHub interface {
	StartGame(roomID string) error
	RestartGame(roomID string) error
	ForceRemove(playerID, roomID string)
}

func NewHandler(rooms *RoomService, hub GameHub, userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository, jwtSecret string) *Handler {
	return &Handler{
		rooms:       rooms,
		gameHub:     hub,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		jwtSecret:   jwtSecret,
	}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/rooms", h.listRooms)
	api.Get("/rooms/:id", h.getRoom)
	api.Get("/me", h.me)
	api.Post("/rooms", h.createRoom)
	api.Post("/rooms/:id/join", h.joinRoom)
	api.Post("/rooms/join/code", h.joinByCode)
	api.Post("/rooms/:id/start", h.startGame)
	api.Post("/rooms/:id/restart", h.restartGame)
	api.Post("/rooms/:id/leave", h.leaveRoom)
}

// resolvePlayer validates the JWT from Authorization header and returns the
// caller's fixed player_id (creating the user record on first login).
func (h *Handler) resolvePlayer(c *fiber.Ctx) (string, error) {
	tokenStr := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	authID, displayName, err := ValidateJWT(tokenStr, h.jwtSecret)
	if err != nil {
		return "", err
	}
	return h.userRepo.GetOrCreate(c.Context(), authID, displayName)
}

// checkActiveSession returns true and writes a 409 response if the player is already in a live room.
// Cleans up stale Redis entries if the room no longer exists.
func (h *Handler) checkActiveSession(c *fiber.Ctx, playerID string) bool {
	if h.sessionRepo == nil {
		return false
	}
	existingRoomID, err := h.sessionRepo.Get(c.Context(), playerID)
	if err != nil || existingRoomID == "" {
		return false
	}
	if _, err := h.rooms.GetByID(existingRoomID); err == nil {
		_ = c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "already_in_room",
			"room_id": existingRoomID,
		})
		return true
	}
	// Room gone — stale session, clean up silently.
	_ = h.sessionRepo.Delete(c.Context(), playerID)
	return false
}

func (h *Handler) me(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	return c.JSON(fiber.Map{"player_id": playerID})
}

func (h *Handler) listRooms(c *fiber.Ctx) error {
	rooms := h.rooms.ListPublic()
	result := make([]dto.RoomResponse, 0, len(rooms))
	for _, r := range rooms {
		result = append(result, ToRoomResponse(r))
	}
	return c.JSON(result)
}

func (h *Handler) getRoom(c *fiber.Ctx) error {
	room, err := h.rooms.GetByID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(ToRoomResponse(room))
}

func (h *Handler) createRoom(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if h.checkActiveSession(c, playerID) {
		return nil
	}

	var req dto.CreateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	hostName := c.Get("X-Player-Name", "방장")
	room, err := h.rooms.Create(req, playerID, hostName)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if h.sessionRepo != nil {
		_ = h.sessionRepo.Set(c.Context(), playerID, room.ID)
	}
	return c.Status(fiber.StatusCreated).JSON(dto.JoinRoomResponse{
		RoomResponse: ToRoomResponse(room),
		PlayerID:     playerID,
	})
}

func (h *Handler) joinRoom(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if h.checkActiveSession(c, playerID) {
		return nil
	}

	var req dto.JoinRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	room, err := h.rooms.Join(c.Params("id"), playerID, req.PlayerName)
	if err != nil {
		status := fiber.StatusConflict
		if err.Error() == "room not found" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	if h.sessionRepo != nil {
		_ = h.sessionRepo.Set(c.Context(), playerID, room.ID)
	}
	return c.JSON(dto.JoinRoomResponse{
		RoomResponse: ToRoomResponse(room),
		PlayerID:     playerID,
	})
}

func (h *Handler) joinByCode(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if h.checkActiveSession(c, playerID) {
		return nil
	}

	var req dto.JoinByCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	room, err := h.rooms.JoinByCode(req.Code, playerID, req.PlayerName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if h.sessionRepo != nil {
		_ = h.sessionRepo.Set(c.Context(), playerID, room.ID)
	}
	return c.JSON(dto.JoinRoomResponse{
		RoomResponse: ToRoomResponse(room),
		PlayerID:     playerID,
	})
}

func (h *Handler) startGame(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	roomID := c.Params("id")
	room, err := h.rooms.GetByID(roomID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if room.HostID != playerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only the host can start the game"})
	}
	if room.GetStatus() != entity.RoomStatusWaiting {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "game already started"})
	}
	if err := h.gameHub.StartGame(roomID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "game started"})
}

func (h *Handler) restartGame(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	roomID := c.Params("id")
	room, err := h.rooms.GetByID(roomID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if room.HostID != playerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only the host can restart the game"})
	}
	if err := h.gameHub.RestartGame(roomID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "game restarted"})
}

type leaveRequest struct {
	PlayerID string `json:"player_id"`
}

// leaveRoom is called by navigator.sendBeacon on pagehide.
// No JWT auth — uses player_id from request body (sendBeacon cannot set custom headers).
func (h *Handler) leaveRoom(c *fiber.Ctx) error {
	var req leaveRequest
	if err := c.BodyParser(&req); err != nil || req.PlayerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "player_id required"})
	}
	roomID := c.Params("id")
	h.gameHub.ForceRemove(req.PlayerID, roomID)
	if h.sessionRepo != nil {
		_ = h.sessionRepo.Delete(c.Context(), req.PlayerID)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
