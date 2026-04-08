package platform

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
)

type Handler struct {
	rooms    *RoomService
	gameHub  GameHub
}

// GameHub is implemented by ws.Hub; defined here to avoid circular imports.
type GameHub interface {
	StartGame(roomID string) error
	RestartGame(roomID string) error
}

func NewHandler(rooms *RoomService, hub GameHub) *Handler {
	return &Handler{rooms: rooms, gameHub: hub}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Post("/rooms", h.createRoom)
	api.Get("/rooms", h.listRooms)
	api.Get("/rooms/:id", h.getRoom)
	api.Post("/rooms/:id/join", h.joinRoom)
	api.Post("/rooms/join/code", h.joinByCode)
	api.Post("/rooms/:id/start", h.startGame)
	api.Post("/rooms/:id/restart", h.restartGame)
}

func (h *Handler) createRoom(c *fiber.Ctx) error {
	var req dto.CreateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	hostID := uuid.NewString()
	hostName := c.Get("X-Player-Name", "방장")

	room, err := h.rooms.Create(req, hostID, hostName)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	resp := dto.JoinRoomResponse{
		RoomResponse: ToRoomResponse(room),
		PlayerID:     hostID,
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
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

func (h *Handler) joinRoom(c *fiber.Ctx) error {
	var req dto.JoinRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	playerID := uuid.NewString()
	room, err := h.rooms.Join(c.Params("id"), playerID, req.PlayerName)
	if err != nil {
		status := fiber.StatusConflict
		if err.Error() == "room not found" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(dto.JoinRoomResponse{
		RoomResponse: ToRoomResponse(room),
		PlayerID:     playerID,
	})
}

func (h *Handler) joinByCode(c *fiber.Ctx) error {
	var req dto.JoinByCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	playerID := uuid.NewString()
	room, err := h.rooms.JoinByCode(req.Code, playerID, req.PlayerName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(dto.JoinRoomResponse{
		RoomResponse: ToRoomResponse(room),
		PlayerID:     playerID,
	})
}

func (h *Handler) startGame(c *fiber.Ctx) error {
	roomID := c.Params("id")
	playerID := c.Get("X-Player-ID")

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
	roomID := c.Params("id")
	playerID := c.Get("X-Player-ID")

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
