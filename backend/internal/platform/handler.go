package platform

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
	"ai-playground/internal/repository"
)

// respondPlayerErr writes the appropriate HTTP error response for resolvePlayer /
// resolvePlayerFull failures. Server-side failures (nil userRepo, DB outages,
// other infrastructure errors) return 500; JWT/auth failures return 401.
// Callers MUST wrap non-auth errors with fiber.ErrInternalServerError (e.g. via
// fmt.Errorf("%w: ...", fiber.ErrInternalServerError, cause)) so errors.As can
// classify them here.
func respondPlayerErr(c *fiber.Ctx, err error) error {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) && fiberErr.Code == fiber.StatusInternalServerError {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
}

// UserStore is the subset of repository.UserRepository used by Handler.
type UserStore interface {
	GetOrCreate(ctx context.Context, authID, displayName string) (string, error)
	GetDisplayName(ctx context.Context, playerID string) (string, error)
	UpdateDisplayName(ctx context.Context, playerID, displayName string) error
}

// GameResultStore is the subset of repository.GameResultRepository used by Handler.
type GameResultStore interface {
	GetStatsByPlayerID(ctx context.Context, playerID string) (repository.PlayerStats, error)
	GetRecentGamesByPlayerID(ctx context.Context, playerID string, limit int) ([]repository.PlayerGameRecord, error)
}

type Handler struct {
	rooms          *RoomService
	gameHub        GameHub
	userRepo       UserStore
	sessionRepo    *repository.SessionRepository
	gameResultRepo GameResultStore
	jwtPublicKey   *ecdsa.PublicKey
}

// GameHub is implemented by ws.Hub; defined here to avoid circular imports.
type GameHub interface {
	StartGame(roomID string) error
	RestartGame(roomID string) error
	ForceRemove(playerID, roomID string)
}

func NewHandler(
	rooms *RoomService,
	hub GameHub,
	userRepo UserStore,
	sessionRepo *repository.SessionRepository,
	gameResultRepo GameResultStore,
	jwtPublicKey *ecdsa.PublicKey,
) *Handler {
	return &Handler{
		rooms:          rooms,
		gameHub:        hub,
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		gameResultRepo: gameResultRepo,
		jwtPublicKey:   jwtPublicKey,
	}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/rooms", h.listRooms)
	api.Get("/rooms/:id", h.getRoom)
	api.Get("/me", h.me)
	api.Put("/me", h.updateMe)
	api.Get("/me/stats", h.myStats)
	api.Get("/me/games", h.myGames)
	api.Post("/rooms", h.createRoom)
	api.Post("/rooms/quick", h.quickMatch)
	api.Post("/rooms/:id/join", h.joinRoom)
	api.Post("/rooms/join/code", h.joinByCode)
	api.Post("/rooms/:id/start", h.startGame)
	api.Post("/rooms/:id/restart", h.restartGame)
	api.Post("/rooms/:id/leave", h.leaveRoom)
}

// resolvePlayer validates the JWT and returns the caller's player_id.
func (h *Handler) resolvePlayer(c *fiber.Ctx) (string, error) {
	tokenStr := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	authID, displayName, err := ValidateJWT(tokenStr, h.jwtPublicKey)
	if err != nil {
		return "", err
	}
	if h.userRepo == nil {
		return "", fiber.ErrInternalServerError
	}
	playerID, err := h.userRepo.GetOrCreate(c.Context(), authID, displayName)
	if err != nil {
		return "", fmt.Errorf("%w: GetOrCreate: %v", fiber.ErrInternalServerError, err)
	}
	return playerID, nil
}

// resolvePlayerFull validates the JWT and returns playerID + stored display_name.
// Used for room entry so the stored (possibly custom) nickname is used, not the JWT name.
func (h *Handler) resolvePlayerFull(c *fiber.Ctx) (playerID, displayName string, err error) {
	tokenStr := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	authID, jwtName, err := ValidateJWT(tokenStr, h.jwtPublicKey)
	if err != nil {
		return "", "", err
	}
	if h.userRepo == nil {
		return "", "", fiber.ErrInternalServerError
	}
	playerID, err = h.userRepo.GetOrCreate(c.Context(), authID, jwtName)
	if err != nil {
		return "", "", fmt.Errorf("%w: GetOrCreate: %v", fiber.ErrInternalServerError, err)
	}
	displayName, err = h.userRepo.GetDisplayName(c.Context(), playerID)
	if err != nil || displayName == "" {
		displayName = jwtName // fallback to JWT name if DB lookup fails
		err = nil
	}
	return playerID, displayName, nil
}

// checkActiveSession returns true and writes a 409 response if the player is already in a live room.
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
	_ = h.sessionRepo.Delete(c.Context(), playerID)
	return false
}

func (h *Handler) me(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}
	displayName, _ := h.userRepo.GetDisplayName(c.Context(), playerID)
	return c.JSON(fiber.Map{"player_id": playerID, "display_name": displayName})
}

type updateMeRequest struct {
	DisplayName string `json:"display_name"`
}

func (h *Handler) updateMe(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}
	var req updateMeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	name := strings.TrimSpace(req.DisplayName)
	if name == "" || len([]rune(name)) > 50 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid display_name"})
	}
	if err := h.userRepo.UpdateDisplayName(c.Context(), playerID, name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"player_id": playerID, "display_name": name})
}

func (h *Handler) myStats(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}
	if h.gameResultRepo == nil {
		return c.JSON(fiber.Map{
			"total_games": 0, "wins": 0, "losses": 0, "win_rate": 0, "by_role": fiber.Map{},
		})
	}
	stats, err := h.gameResultRepo.GetStatsByPlayerID(c.Context(), playerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	byRole := fiber.Map{}
	for role, rs := range stats.ByRole {
		byRole[role] = fiber.Map{
			"games":    rs.Games,
			"wins":     rs.Wins,
			"win_rate": rs.WinRate,
		}
	}
	return c.JSON(fiber.Map{
		"total_games": stats.TotalGames,
		"wins":        stats.Wins,
		"losses":      stats.Losses,
		"win_rate":    stats.WinRate,
		"by_role":     byRole,
	})
}

func (h *Handler) myGames(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}
	limit := c.QueryInt("limit", 20)
	if h.gameResultRepo == nil {
		return c.JSON([]fiber.Map{})
	}
	records, err := h.gameResultRepo.GetRecentGamesByPlayerID(c.Context(), playerID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	result := make([]fiber.Map, 0, len(records))
	for _, r := range records {
		result = append(result, fiber.Map{
			"game_id":      r.GameID,
			"played_at":    r.PlayedAt,
			"role":         r.Role,
			"survived":     r.Survived,
			"won":          r.Won,
			"round_count":  r.RoundCount,
			"duration_sec": r.DurationSec,
		})
	}
	return c.JSON(result)
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
	playerID, displayName, err := h.resolvePlayerFull(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}
	if h.checkActiveSession(c, playerID) {
		return nil
	}
	var req dto.CreateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	room, err := h.rooms.Create(req, playerID, displayName)
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
	playerID, displayName, err := h.resolvePlayerFull(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}
	if h.checkActiveSession(c, playerID) {
		return nil
	}
	room, err := h.rooms.Join(c.Params("id"), playerID, displayName)
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

func (h *Handler) quickMatch(c *fiber.Ctx) error {
	playerID, displayName, err := h.resolvePlayerFull(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}

	room, created, err := h.rooms.FindOrCreatePublicRoom(playerID, displayName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"room_id":   room.ID,
		"player_id": playerID,
		"created":   created,
	})
}

func (h *Handler) joinByCode(c *fiber.Ctx) error {
	playerID, displayName, err := h.resolvePlayerFull(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}
	if h.checkActiveSession(c, playerID) {
		return nil
	}
	var req dto.JoinByCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	room, err := h.rooms.JoinByCode(req.Code, playerID, displayName)
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
		return respondPlayerErr(c, err)
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
		return respondPlayerErr(c, err)
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
// No JWT auth — uses player_id from request body.
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
