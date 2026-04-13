# User Profile & Stats Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Google 계정 1:1 고정 닉네임, `/profile` 페이지(닉네임 수정·통계·최근 게임 기록)를 구현하고 방 입장 시 닉네임 입력을 제거한다.

**Architecture:** 백엔드는 `UserRepository`에 닉네임 수정/조회를, `GameResultRepository`에 통계 집계를 추가한다. `Handler`에 `PUT /api/me`, `GET /api/me/stats`, `GET /api/me/games` 를 추가하고, `createRoom`/`joinRoom`/`joinByCode`는 users 테이블의 display_name을 자동으로 사용한다. 프론트엔드는 `authStore`에 `displayName`을 추가하고 `ProfilePage`를 신규 생성한다.

**Tech Stack:** Go, Fiber, PostgreSQL, pgx, React, TypeScript, Zustand, @supabase/supabase-js

---

## File Map

| 파일 | 변경 | 책임 |
|------|------|------|
| `backend/internal/repository/user.go` | 수정 | GetOrCreate display_name 보존, UpdateDisplayName, GetDisplayName 추가 |
| `backend/internal/repository/room.go` | 수정 | GameResultRepository에 PlayerStats, PlayerGameRecord, GetStatsByPlayerID, GetRecentGamesByPlayerID 추가 |
| `backend/internal/platform/handler.go` | 수정 | gameResultRepo 필드, resolvePlayerFull, PUT /api/me, GET /api/me/stats, GET /api/me/games, createRoom/joinRoom/joinByCode display_name 자동 사용 |
| `backend/internal/platform/handler_test.go` | 수정 | setupApp gameResultRepo nil 추가, 새 엔드포인트 401 테스트 |
| `backend/cmd/server/main.go` | 수정 | NewHandler에 gameResultRepo 전달 |
| `frontend/src/store/authStore.ts` | 수정 | displayName 필드, initialize/onAuthStateChange에서 display_name 저장 |
| `frontend/src/api.ts` | 수정 | player_name 파라미터 제거, updateMe/getMyStats/getMyGames 추가 |
| `frontend/src/pages/LobbyPage.tsx` | 수정 | 닉네임 입력 제거, 프로필 진입 버튼 추가 |
| `frontend/src/main.tsx` | 수정 | /profile 라우트 추가 |
| `frontend/src/pages/ProfilePage.tsx` | 신규 | 프로필 페이지 전체 |

---

## Task 1: 백엔드 — UserRepository 확장

**Files:**
- Modify: `backend/internal/repository/user.go`

- [ ] **Step 1: user.go 전체 교체**

`backend/internal/repository/user.go`를 다음으로 교체:

```go
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// GetOrCreate returns the player_id for auth_id. On first call it creates a new
// user with a fresh UUID and sets display_name from Google JWT.
// On subsequent calls it does NOT overwrite display_name (preserving custom nicknames).
func (r *UserRepository) GetOrCreate(ctx context.Context, authID, displayName string) (string, error) {
	newPlayerID := uuid.NewString()
	var playerID string
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (auth_id, player_id, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (auth_id) DO UPDATE SET player_id = users.player_id
		RETURNING player_id
	`, authID, newPlayerID, displayName).Scan(&playerID)
	return playerID, err
}

// GetByAuthID returns the player_id for an existing user.
// Returns empty string (no error) if the user does not exist.
func (r *UserRepository) GetByAuthID(ctx context.Context, authID string) (string, error) {
	var playerID string
	err := r.db.QueryRow(ctx,
		`SELECT player_id FROM users WHERE auth_id = $1`, authID,
	).Scan(&playerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return playerID, err
}

// GetDisplayName returns the display_name for a player_id.
// Returns empty string (no error) if the user does not exist.
func (r *UserRepository) GetDisplayName(ctx context.Context, playerID string) (string, error) {
	var name string
	err := r.db.QueryRow(ctx,
		`SELECT display_name FROM users WHERE player_id = $1`, playerID,
	).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

// UpdateDisplayName sets a new display_name for the given player_id.
func (r *UserRepository) UpdateDisplayName(ctx context.Context, playerID, displayName string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET display_name = $1 WHERE player_id = $2`,
		displayName, playerID,
	)
	return err
}
```

- [ ] **Step 2: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/repository/user.go
git commit -m "feat: preserve display_name on re-login, add GetDisplayName and UpdateDisplayName"
```

---

## Task 2: 백엔드 — GameResultRepository 통계 메서드

**Files:**
- Modify: `backend/internal/repository/room.go`

- [ ] **Step 1: room.go 하단에 타입 + 메서드 추가**

`backend/internal/repository/room.go`의 맨 끝에 다음을 추가:

```go
// ─── Stats ───────────────────────────────────────────────────────────────────

type RoleStats struct {
	Games   int
	Wins    int
	WinRate float64
}

type PlayerStats struct {
	TotalGames int
	Wins       int
	Losses     int
	WinRate    float64
	ByRole     map[string]RoleStats
}

type PlayerGameRecord struct {
	GameID      string
	PlayedAt    string // RFC3339
	Role        string
	Survived    bool
	Won         bool
	RoundCount  int
	DurationSec int
}

// GetStatsByPlayerID aggregates win/loss stats for a human player.
func (r *GameResultRepository) GetStatsByPlayerID(ctx context.Context, playerID string) (PlayerStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			p.role,
			COUNT(*) AS games,
			COUNT(*) FILTER (WHERE
				(p.role = 'mafia' AND gr.winner_team = 'mafia') OR
				(p.role != 'mafia' AND gr.winner_team = 'citizen')
			) AS wins
		FROM game_result_players p
		JOIN game_results gr ON p.game_result_id = gr.id
		WHERE p.player_id = $1 AND p.is_ai = false
		GROUP BY p.role
	`, playerID)
	if err != nil {
		return PlayerStats{}, err
	}
	defer rows.Close()

	byRole := make(map[string]RoleStats)
	for rows.Next() {
		var role string
		var games, wins int
		if err := rows.Scan(&role, &games, &wins); err != nil {
			return PlayerStats{}, err
		}
		wr := 0.0
		if games > 0 {
			wr = float64(wins) / float64(games)
		}
		byRole[role] = RoleStats{Games: games, Wins: wins, WinRate: wr}
	}
	if err := rows.Err(); err != nil {
		return PlayerStats{}, err
	}

	var total, wins int
	for _, rs := range byRole {
		total += rs.Games
		wins += rs.Wins
	}
	wr := 0.0
	if total > 0 {
		wr = float64(wins) / float64(total)
	}
	return PlayerStats{
		TotalGames: total,
		Wins:       wins,
		Losses:     total - wins,
		WinRate:    wr,
		ByRole:     byRole,
	}, nil
}

// GetRecentGamesByPlayerID returns the most recent game records for a player.
func (r *GameResultRepository) GetRecentGamesByPlayerID(ctx context.Context, playerID string, limit int) ([]PlayerGameRecord, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			gr.id,
			gr.created_at,
			p.role,
			p.survived,
			(p.role = 'mafia' AND gr.winner_team = 'mafia') OR
			(p.role != 'mafia' AND gr.winner_team = 'citizen') AS won,
			gr.round_count,
			gr.duration_sec
		FROM game_result_players p
		JOIN game_results gr ON p.game_result_id = gr.id
		WHERE p.player_id = $1 AND p.is_ai = false
		ORDER BY gr.created_at DESC
		LIMIT $2
	`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []PlayerGameRecord
	for rows.Next() {
		var rec PlayerGameRecord
		var playedAt interface{ String() string }
		if err := rows.Scan(
			&rec.GameID, &playedAt, &rec.Role,
			&rec.Survived, &rec.Won,
			&rec.RoundCount, &rec.DurationSec,
		); err != nil {
			return nil, err
		}
		rec.PlayedAt = playedAt.String()
		records = append(records, rec)
	}
	return records, rows.Err()
}
```

- [ ] **Step 2: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/repository/room.go
git commit -m "feat: add GetStatsByPlayerID and GetRecentGamesByPlayerID to GameResultRepository"
```

---

## Task 3: 백엔드 — Handler 확장

**Files:**
- Modify: `backend/internal/platform/handler.go`

- [ ] **Step 1: handler.go 전체 교체**

`backend/internal/platform/handler.go`를 다음으로 교체:

```go
package platform

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
	"ai-playground/internal/repository"
)

type Handler struct {
	rooms          *RoomService
	gameHub        GameHub
	userRepo       *repository.UserRepository
	sessionRepo    *repository.SessionRepository
	gameResultRepo *repository.GameResultRepository
	jwtSecret      string
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
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	gameResultRepo *repository.GameResultRepository,
	jwtSecret string,
) *Handler {
	return &Handler{
		rooms:          rooms,
		gameHub:        hub,
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		gameResultRepo: gameResultRepo,
		jwtSecret:      jwtSecret,
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
	api.Post("/rooms/:id/join", h.joinRoom)
	api.Post("/rooms/join/code", h.joinByCode)
	api.Post("/rooms/:id/start", h.startGame)
	api.Post("/rooms/:id/restart", h.restartGame)
	api.Post("/rooms/:id/leave", h.leaveRoom)
}

// resolvePlayer validates the JWT and returns the caller's player_id.
func (h *Handler) resolvePlayer(c *fiber.Ctx) (string, error) {
	tokenStr := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	authID, displayName, err := ValidateJWT(tokenStr, h.jwtSecret)
	if err != nil {
		return "", err
	}
	return h.userRepo.GetOrCreate(c.Context(), authID, displayName)
}

// resolvePlayerFull validates the JWT and returns playerID + stored display_name.
// Used for room entry so the stored (possibly custom) nickname is used, not the JWT name.
func (h *Handler) resolvePlayerFull(c *fiber.Ctx) (playerID, displayName string, err error) {
	tokenStr := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	authID, jwtName, err := ValidateJWT(tokenStr, h.jwtSecret)
	if err != nil {
		return "", "", err
	}
	playerID, err = h.userRepo.GetOrCreate(c.Context(), authID, jwtName)
	if err != nil {
		return "", "", err
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	displayName := ""
	if h.userRepo != nil {
		displayName, _ = h.userRepo.GetDisplayName(c.Context(), playerID)
	}
	return c.JSON(fiber.Map{"player_id": playerID, "display_name": displayName})
}

type updateMeRequest struct {
	DisplayName string `json:"display_name"`
}

func (h *Handler) updateMe(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var req updateMeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	name := strings.TrimSpace(req.DisplayName)
	if name == "" || len([]rune(name)) > 50 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid display_name"})
	}
	if h.userRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "not available"})
	}
	if err := h.userRepo.UpdateDisplayName(c.Context(), playerID, name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"player_id": playerID, "display_name": name})
}

func (h *Handler) myStats(c *fiber.Ctx) error {
	playerID, err := h.resolvePlayer(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
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

func (h *Handler) joinByCode(c *fiber.Ctx) error {
	playerID, displayName, err := h.resolvePlayerFull(c)
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
```

- [ ] **Step 2: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: `main.go`에서 `NewHandler` 인수 불일치 컴파일 에러 발생 — Task 5에서 수정

- [ ] **Step 3: 커밋 (빌드 에러 있어도 커밋)**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/platform/handler.go
git commit -m "feat: add profile endpoints, resolvePlayerFull for auto display_name in room entry"
```

---

## Task 4: 백엔드 — handler_test.go 업데이트

**Files:**
- Modify: `backend/internal/platform/handler_test.go`

- [ ] **Step 1: setupApp에 gameResultRepo nil 추가**

`handler_test.go`의 `setupApp` 함수에서 `NewHandler` 호출 부분을 찾아 수정:

현재:
```go
h := NewHandler(svc, &mockHub{}, nil, nil, "")
```

교체:
```go
h := NewHandler(svc, &mockHub{}, nil, nil, nil, "")
```

- [ ] **Step 2: 새 엔드포인트 401 테스트 추가**

`handler_test.go` 맨 끝, `// ─── helpers` 섹션 바로 위에 다음 테스트 추가:

```go
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
```

- [ ] **Step 3: 테스트 실행**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./internal/platform/ -v
```

Expected: 모든 테스트 PASS (빌드 에러는 main.go 때문 — Task 5에서 해결)

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/platform/handler_test.go
git commit -m "test: update handler_test for new NewHandler signature, add 401 tests for profile endpoints"
```

---

## Task 5: 백엔드 — main.go NewHandler 배선 수정

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: NewHandler 호출에 gameResultRepo 추가**

`backend/cmd/server/main.go`에서 다음 줄을 찾아:

```go
handler := platform.NewHandler(roomSvc, gameHub, userRepo, sessionRepo, cfg.Supabase.JWTSecret)
```

다음으로 교체:

```go
handler := platform.NewHandler(roomSvc, gameHub, userRepo, sessionRepo, gameResultRepo, cfg.Supabase.JWTSecret)
```

- [ ] **Step 2: 빌드 + 테스트 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./... && go test ./...
```

Expected: 빌드 성공, 모든 테스트 PASS

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/cmd/server/main.go
git commit -m "feat: wire gameResultRepo into Handler for stats endpoints"
```

---

## Task 6: 프론트엔드 — authStore displayName 추가

**Files:**
- Modify: `frontend/src/store/authStore.ts`

- [ ] **Step 1: authStore.ts 전체 교체**

`frontend/src/store/authStore.ts`를 다음으로 교체:

```typescript
import { create } from 'zustand'
import type { User } from '@supabase/supabase-js'
import { supabase } from '../lib/supabase'

interface AuthStore {
  user: User | null
  playerID: string
  displayName: string
  loading: boolean
  initialize: () => Promise<void>
  signInWithGoogle: () => Promise<void>
  signOut: () => Promise<void>
  getAccessToken: () => Promise<string>
}

let initialized = false

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  playerID: '',
  displayName: '',
  loading: true,

  async initialize() {
    if (initialized) return
    initialized = true

    const { data: { session } } = await supabase.auth.getSession()
    if (session?.user) {
      const res = await fetch('/api/me', {
        headers: { Authorization: `Bearer ${session.access_token}` },
      })
      if (res.ok) {
        const data = await res.json() as { player_id: string; display_name: string }
        set({ user: session.user, playerID: data.player_id, displayName: data.display_name, loading: false })
      } else {
        set({ user: session.user, loading: false })
      }
    } else {
      set({ loading: false })
    }

    supabase.auth.onAuthStateChange(async (event, session) => {
      if (event === 'SIGNED_IN' && session?.user) {
        const res = await fetch('/api/me', {
          headers: { Authorization: `Bearer ${session.access_token}` },
        })
        if (res.ok) {
          const data = await res.json() as { player_id: string; display_name: string }
          set({ user: session.user, playerID: data.player_id, displayName: data.display_name })
        } else {
          set({ user: session.user })
        }
      } else if (event === 'SIGNED_OUT') {
        set({ user: null, playerID: '', displayName: '' })
      }
    })
  },

  async signInWithGoogle() {
    await supabase.auth.signInWithOAuth({
      provider: 'google',
      options: { redirectTo: `${window.location.origin}/lobby` },
    })
  },

  async signOut() {
    await supabase.auth.signOut()
    set({ user: null, playerID: '', displayName: '' })
  },

  async getAccessToken() {
    const { data: { session } } = await supabase.auth.getSession()
    return session?.access_token ?? ''
  },
}))
```

- [ ] **Step 2: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/store/authStore.ts
git commit -m "feat: add displayName field to authStore"
```

---

## Task 7: 프론트엔드 — api.ts player_name 제거 + 프로필 API 추가

**Files:**
- Modify: `frontend/src/api.ts`

- [ ] **Step 1: api.ts 전체 교체**

`frontend/src/api.ts`를 다음으로 교체:

```typescript
import { supabase } from './lib/supabase'

const BASE = '/api'

async function getAuthHeader(): Promise<Record<string, string>> {
  const { data: { session } } = await supabase.auth.getSession()
  if (session?.access_token) {
    return { Authorization: `Bearer ${session.access_token}` }
  }
  return {}
}

async function request<T>(path: string, options?: RequestInit & { headers?: Record<string, string> }): Promise<T> {
  const authHeader = await getAuthHeader()
  const { headers: optHeaders, ...restOptions } = options ?? {}
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...authHeader, ...optHeaders },
    ...restOptions,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as { error?: string; room_id?: string }
    const err = new Error(body.error ?? `HTTP ${res.status}`) as Error & { roomID?: string }
    if (res.status === 409 && body.room_id) {
      err.roomID = body.room_id
    }
    throw err
  }
  return res.json() as Promise<T>
}

export interface CreateRoomParams {
  name: string
  visibility: 'public' | 'private'
  max_humans?: number
}

export interface JoinRoomParams {
  room_id: string
}

export interface JoinByCodeParams {
  code: string
}

export interface JoinResponse {
  player_id: string
  id: string
}

export interface RoleStats {
  games: number
  wins: number
  win_rate: number
}

export interface MyStatsResponse {
  total_games: number
  wins: number
  losses: number
  win_rate: number
  by_role: Record<string, RoleStats>
}

export interface MyGameRecord {
  game_id: string
  played_at: string
  role: string
  survived: boolean
  won: boolean
  round_count: number
  duration_sec: number
}

export function listRooms() {
  return request<import('./types').Room[]>('/rooms')
}

export function createRoom(params: CreateRoomParams) {
  return request<JoinResponse>('/rooms', {
    method: 'POST',
    body: JSON.stringify({
      name: params.name,
      visibility: params.visibility,
      max_humans: params.max_humans ?? 6,
    }),
  })
}

export function joinRoom(params: JoinRoomParams) {
  return request<JoinResponse>(`/rooms/${params.room_id}/join`, {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export function joinByCode(params: JoinByCodeParams) {
  return request<JoinResponse>('/rooms/join/code', {
    method: 'POST',
    body: JSON.stringify({ code: params.code }),
  })
}

export function startGame(roomID: string) {
  return request<void>(`/rooms/${roomID}/start`, { method: 'POST' })
}

export function restartGame(roomID: string) {
  return request<void>(`/rooms/${roomID}/restart`, { method: 'POST' })
}

export function updateMe(displayName: string) {
  return request<{ player_id: string; display_name: string }>('/me', {
    method: 'PUT',
    body: JSON.stringify({ display_name: displayName }),
  })
}

export function getMyStats() {
  return request<MyStatsResponse>('/me/stats')
}

export function getMyGames(limit = 20) {
  return request<MyGameRecord[]>(`/me/games?limit=${limit}`)
}
```

- [ ] **Step 2: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build 2>&1 | grep -E "error|Error"
```

Expected: LobbyPage.tsx에서 `player_name` 관련 타입 에러 발생 — Task 8에서 수정. 다른 에러 없어야 함.

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/api.ts
git commit -m "feat: remove player_name from room entry API, add updateMe/getMyStats/getMyGames"
```

---

## Task 8: 프론트엔드 — LobbyPage 닉네임 입력 제거 + 프로필 버튼

**Files:**
- Modify: `frontend/src/pages/LobbyPage.tsx`

- [ ] **Step 1: import + authStore 추가**

파일 상단 import 목록에서 `useEffect, useRef, useState, type CSSProperties`가 있는 줄을 찾아 `useAuthStore` import 추가:

```typescript
import { useAuthStore } from '../store/authStore'
```

컴포넌트 함수 내부, `const navigate = useNavigate()` 바로 아래에 추가:

```typescript
const { displayName } = useAuthStore()
```

- [ ] **Step 2: 닉네임 state 제거**

다음 useState 3개를 찾아 삭제:

```typescript
const [joinName, setJoinName] = useState('')
```
```typescript
const [createPlayerName, setCreatePlayerName] = useState('')
```
```typescript
const [codePlayerName, setCodePlayerName] = useState('')
```

- [ ] **Step 3: handleJoinRoom 수정**

현재:
```typescript
async function handleJoinRoom() {
  if (!joiningRoom || !joinName.trim()) return
  setJoinError('')
  try {
    const res = await joinRoom({ room_id: joiningRoom.id, player_name: joinName.trim() })
```

교체:
```typescript
async function handleJoinRoom() {
  if (!joiningRoom) return
  setJoinError('')
  try {
    const res = await joinRoom({ room_id: joiningRoom.id })
```

- [ ] **Step 4: handleCreateRoom 수정**

현재:
```typescript
async function handleCreateRoom() {
  if (!createName.trim() || !createPlayerName.trim()) return
  setCreateError('')
  try {
    const res = await createRoom({
      name: createName.trim(),
      visibility: createVisibility,
      player_name: createPlayerName.trim(),
    })
```

교체:
```typescript
async function handleCreateRoom() {
  if (!createName.trim()) return
  setCreateError('')
  try {
    const res = await createRoom({
      name: createName.trim(),
      visibility: createVisibility,
    })
```

- [ ] **Step 5: handleJoinByCode 수정**

현재:
```typescript
async function handleJoinByCode() {
  if (!codeInput.trim() || !codePlayerName.trim()) return
  setCodeError('')
  try {
    const res = await joinByCode({ code: codeInput.trim(), player_name: codePlayerName.trim() })
```

교체:
```typescript
async function handleJoinByCode() {
  if (!codeInput.trim()) return
  setCodeError('')
  try {
    const res = await joinByCode({ code: codeInput.trim() })
```

- [ ] **Step 6: JSX에서 닉네임 input 3개 제거**

**방 만들기 섹션** (lines 258-264 근처)에서 닉네임 input 제거:

```tsx
<input
  style={inputSt}
  placeholder="닉네임"
  value={createPlayerName}
  onChange={(e) => setCreatePlayerName(e.target.value)}
  onKeyDown={(e) => e.key === 'Enter' && handleCreateRoom()}
/>
```
위 블록 전체 삭제.

**코드 참가 섹션** (lines 317-323 근처)에서 닉네임 input 제거:

```tsx
<input
  style={inputSt}
  placeholder="닉네임"
  value={codePlayerName}
  onChange={(e) => setCodePlayerName(e.target.value)}
  onKeyDown={(e) => e.key === 'Enter' && handleJoinByCode()}
/>
```
위 블록 전체 삭제.

**Join 모달** (lines 517-524 근처)에서 닉네임 input 제거:

```tsx
<input
  style={inputSt}
  placeholder="닉네임을 입력하세요"
  value={joinName}
  onChange={(e) => setJoinName(e.target.value)}
  onKeyDown={(e) => e.key === 'Enter' && handleJoinRoom()}
  autoFocus
/>
```
위 블록 전체 삭제.

- [ ] **Step 7: Nav에 프로필 버튼 추가**

`<nav>` 내부에서 `LOBBY` 텍스트를 표시하는 `<div>` 바로 오른쪽에 프로필 버튼 추가. 현재:

```tsx
<div style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted, letterSpacing: '0.1em', textTransform: 'uppercase' }}>
  LOBBY
</div>
```

이 `<div>` 전체를 다음으로 교체 (LOBBY 텍스트와 프로필 버튼을 함께 배치):

```tsx
<div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
  <div style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted, letterSpacing: '0.1em', textTransform: 'uppercase' }}>
    LOBBY
  </div>
  <button
    onClick={() => navigate('/profile')}
    style={{
      fontFamily: MONO, fontSize: '11px', color: T.textMuted,
      background: 'none', border: `1px solid ${T.surfaceBorder}`,
      borderRadius: '2px', padding: '5px 12px', cursor: 'pointer',
      letterSpacing: '0.08em', textTransform: 'uppercase',
      transition: 'all 100ms ease',
    }}
  >
    {displayName || 'PROFILE'}
  </button>
</div>
```

- [ ] **Step 8: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 9: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/pages/LobbyPage.tsx
git commit -m "feat: remove per-room nickname input, add profile nav button to LobbyPage"
```

---

## Task 9: 프론트엔드 — /profile 라우트 추가

**Files:**
- Modify: `frontend/src/main.tsx`

- [ ] **Step 1: main.tsx 수정**

`frontend/src/main.tsx`에서 ProfilePage import + 라우트 추가:

```tsx
import { StrictMode, useEffect, type ReactNode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import './index.css'
import LandingPage from './pages/LandingPage'
import LobbyPage from './pages/LobbyPage'
import RoomPage from './pages/RoomPage'
import ProfilePage from './pages/ProfilePage'
import { useAuthStore } from './store/authStore'

function AppInit({ children }: { children: ReactNode }) {
  useEffect(() => {
    void useAuthStore.getState().initialize()
  }, [])
  return <>{children}</>
}

const router = createBrowserRouter([
  { path: '/', element: <LandingPage /> },
  { path: '/lobby', element: <LobbyPage /> },
  { path: '/rooms/:id', element: <RoomPage /> },
  { path: '/profile', element: <ProfilePage /> },
])

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppInit>
      <RouterProvider router={router} />
    </AppInit>
  </StrictMode>,
)
```

- [ ] **Step 2: 빌드 확인 (ProfilePage.tsx 없어서 에러 발생 예상)**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build 2>&1 | grep -E "error|Error"
```

Expected: `ProfilePage` 모듈 not found 에러 — Task 10에서 해결. 다른 에러 없어야 함.

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/main.tsx
git commit -m "feat: add /profile route to router"
```

---

## Task 10: 프론트엔드 — ProfilePage 생성

**Files:**
- Create: `frontend/src/pages/ProfilePage.tsx`

- [ ] **Step 1: ProfilePage.tsx 생성**

`frontend/src/pages/ProfilePage.tsx`를 다음 내용으로 생성:

```tsx
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { updateMe, getMyStats, getMyGames } from '../api'
import type { MyStatsResponse, MyGameRecord } from '../api'

// ─── Design tokens (same as LobbyPage) ───────────────────────────────────────
const T = {
  bg:            '#0E0C09',
  surface:       '#181410',
  surfaceHigh:   '#221E17',
  surfaceBorder: '#2E2820',
  accent:        '#C4963A',
  accentDim:     'rgba(196,150,58,0.12)',
  text:          '#ECE7DE',
  textMuted:     '#786F62',
  textDim:       '#4A4438',
  danger:        '#8C1F1F',
  dangerDim:     'rgba(140,31,31,0.15)',
  police:        '#3D7FA8',
}
const SERIF = "'Instrument Serif', Georgia, serif"
const SANS  = "'DM Sans', system-ui, sans-serif"
const MONO  = "'JetBrains Mono', monospace"

const ROLE_KO: Record<string, string> = {
  mafia: '마피아',
  citizen: '시민',
  police: '경찰',
}

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return s > 0 ? `${m}분 ${s}초` : `${m}분`
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

export default function ProfilePage() {
  const navigate = useNavigate()
  const { user, playerID, displayName, signOut, loading } = useAuthStore()

  const [editMode, setEditMode] = useState(false)
  const [nameInput, setNameInput] = useState('')
  const [nameError, setNameError] = useState('')
  const [nameSaving, setNameSaving] = useState(false)

  const [stats, setStats] = useState<MyStatsResponse | null>(null)
  const [games, setGames] = useState<MyGameRecord[]>([])
  const [statsLoading, setStatsLoading] = useState(true)

  // Redirect to lobby if not logged in
  useEffect(() => {
    if (!loading && !user) {
      navigate('/lobby')
    }
  }, [user, loading, navigate])

  // Load stats + games
  useEffect(() => {
    if (!playerID) return
    setStatsLoading(true)
    Promise.all([getMyStats(), getMyGames(20)])
      .then(([s, g]) => {
        setStats(s)
        setGames(g ?? [])
      })
      .catch(() => {})
      .finally(() => setStatsLoading(false))
  }, [playerID])

  async function handleSaveName() {
    const trimmed = nameInput.trim()
    if (!trimmed) { setNameError('닉네임을 입력해주세요.'); return }
    if (trimmed.length > 50) { setNameError('50자 이하로 입력해주세요.'); return }
    setNameSaving(true)
    setNameError('')
    try {
      const data = await updateMe(trimmed)
      useAuthStore.setState({ displayName: data.display_name })
      setEditMode(false)
    } catch (e) {
      setNameError(e instanceof Error ? e.message : '저장 실패')
    } finally {
      setNameSaving(false)
    }
  }

  async function handleSignOut() {
    await signOut()
    navigate('/')
  }

  const avatarUrl = user?.user_metadata?.avatar_url as string | undefined
  const email = user?.email ?? ''

  return (
    <div style={{ minHeight: '100dvh', background: T.bg, color: T.text, fontFamily: SANS }}>

      {/* ── Nav ──────────────────────────────────────────────────────────── */}
      <nav style={{
        position: 'fixed', top: 0, left: 0, right: 0, zIndex: 40,
        height: '60px', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '0 32px', background: T.bg,
        borderBottom: `1px solid ${T.surfaceBorder}`,
      }}>
        <button
          onClick={() => navigate('/')}
          style={{ fontFamily: SERIF, fontSize: '20px', color: T.accent, letterSpacing: '-0.02em', background: 'none', border: 'none', cursor: 'pointer', padding: 0 }}
        >
          AI Mafia
        </button>
        <div style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted, letterSpacing: '0.1em', textTransform: 'uppercase' }}>
          PROFILE
        </div>
      </nav>

      {/* ── Content ──────────────────────────────────────────────────────── */}
      <main style={{ maxWidth: '760px', margin: '0 auto', padding: '88px 32px 64px' }}>

        {/* ── Header: avatar + nickname + email ─────────────────────────── */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '20px', marginBottom: '40px' }}>
          {avatarUrl ? (
            <img src={avatarUrl} alt="avatar" style={{ width: '64px', height: '64px', borderRadius: '50%', objectFit: 'cover', border: `1px solid ${T.surfaceBorder}` }} />
          ) : (
            <div style={{ width: '64px', height: '64px', borderRadius: '50%', background: T.surfaceHigh, border: `1px solid ${T.surfaceBorder}`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: MONO, fontSize: '20px', color: T.textMuted }}>
              {(displayName || '?')[0].toUpperCase()}
            </div>
          )}
          <div style={{ flex: 1 }}>
            {/* Nickname row */}
            {editMode ? (
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                <input
                  value={nameInput}
                  onChange={(e) => setNameInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleSaveName()}
                  autoFocus
                  maxLength={50}
                  style={{
                    fontFamily: SANS, fontSize: '22px', color: T.text,
                    background: T.surfaceHigh, border: `1px solid ${T.accent}60`,
                    borderRadius: '2px', padding: '4px 8px', outline: 'none', width: '200px',
                  }}
                />
                <button
                  onClick={handleSaveName}
                  disabled={nameSaving}
                  style={{
                    fontFamily: MONO, fontSize: '11px', color: T.accent,
                    background: T.accentDim, border: `1px solid ${T.accent}50`,
                    borderRadius: '2px', padding: '5px 12px', cursor: 'pointer',
                    textTransform: 'uppercase', letterSpacing: '0.08em',
                  }}
                >
                  {nameSaving ? '저장 중...' : '저장'}
                </button>
                <button
                  onClick={() => { setEditMode(false); setNameError('') }}
                  style={{
                    fontFamily: MONO, fontSize: '11px', color: T.textMuted,
                    background: 'transparent', border: `1px solid ${T.surfaceBorder}`,
                    borderRadius: '2px', padding: '5px 12px', cursor: 'pointer',
                    textTransform: 'uppercase', letterSpacing: '0.08em',
                  }}
                >
                  취소
                </button>
              </div>
            ) : (
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '4px' }}>
                <span style={{ fontFamily: SERIF, fontSize: '26px', letterSpacing: '-0.01em' }}>
                  {displayName || '—'}
                </span>
                <button
                  onClick={() => { setNameInput(displayName); setEditMode(true) }}
                  style={{
                    fontFamily: MONO, fontSize: '10px', color: T.textMuted,
                    background: 'none', border: `1px solid ${T.surfaceBorder}`,
                    borderRadius: '2px', padding: '3px 8px', cursor: 'pointer',
                    textTransform: 'uppercase', letterSpacing: '0.08em',
                  }}
                >
                  수정
                </button>
              </div>
            )}
            {nameError && (
              <span style={{ fontFamily: MONO, fontSize: '11px', color: T.danger, display: 'block', marginBottom: '4px' }}>
                {nameError}
              </span>
            )}
            <span style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted }}>{email}</span>
          </div>
        </div>

        <div style={{ height: '1px', background: T.surfaceBorder, marginBottom: '32px' }} />

        {/* ── Stats cards ───────────────────────────────────────────────── */}
        {statsLoading ? (
          <div style={{ fontFamily: MONO, fontSize: '11px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em', marginBottom: '32px' }}>
            LOADING...
          </div>
        ) : (
          <>
            <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.12em', display: 'block', marginBottom: '12px' }}>
              전체 통계
            </span>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '12px', marginBottom: '32px' }}>
              {[
                { label: '총 게임', value: String(stats?.total_games ?? 0) },
                { label: '승', value: String(stats?.wins ?? 0) },
                { label: '패', value: String(stats?.losses ?? 0) },
                { label: '승률', value: stats && stats.total_games > 0 ? `${(stats.win_rate * 100).toFixed(1)}%` : '—' },
              ].map(({ label, value }) => (
                <div key={label} style={{
                  background: T.surface, border: `1px solid ${T.surfaceBorder}`,
                  borderRadius: '4px', padding: '16px',
                  display: 'flex', flexDirection: 'column', gap: '6px',
                }}>
                  <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.1em' }}>{label}</span>
                  <span style={{ fontFamily: SERIF, fontSize: '28px', color: T.text, letterSpacing: '-0.02em' }}>{value}</span>
                </div>
              ))}
            </div>

            {/* ── Role breakdown ─────────────────────────────────────────── */}
            {stats && Object.keys(stats.by_role).length > 0 && (
              <>
                <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.12em', display: 'block', marginBottom: '12px' }}>
                  역할별 통계
                </span>
                <div style={{
                  background: T.surface, border: `1px solid ${T.surfaceBorder}`,
                  borderRadius: '4px', marginBottom: '32px', overflow: 'hidden',
                }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr style={{ borderBottom: `1px solid ${T.surfaceBorder}` }}>
                        {['역할', '게임', '승', '패', '승률'].map((h) => (
                          <th key={h} style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.1em', padding: '10px 16px', textAlign: 'left', fontWeight: 400 }}>{h}</th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {Object.entries(stats.by_role).map(([role, rs]) => (
                        <tr key={role} style={{ borderBottom: `1px solid ${T.surfaceBorder}` }}>
                          <td style={{ padding: '12px 16px', fontFamily: SANS, fontSize: '13px', color: T.text }}>{ROLE_KO[role] ?? role}</td>
                          <td style={{ padding: '12px 16px', fontFamily: MONO, fontSize: '12px', color: T.textMuted }}>{rs.games}</td>
                          <td style={{ padding: '12px 16px', fontFamily: MONO, fontSize: '12px', color: T.textMuted }}>{rs.wins}</td>
                          <td style={{ padding: '12px 16px', fontFamily: MONO, fontSize: '12px', color: T.textMuted }}>{rs.games - rs.wins}</td>
                          <td style={{ padding: '12px 16px', fontFamily: MONO, fontSize: '12px', color: T.accent }}>{rs.games > 0 ? `${(rs.win_rate * 100).toFixed(1)}%` : '—'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}

            {/* ── Recent games ───────────────────────────────────────────── */}
            <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.12em', display: 'block', marginBottom: '12px' }}>
              최근 게임
            </span>
            {games.length === 0 ? (
              <div style={{ padding: '32px 0', fontFamily: MONO, fontSize: '11px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em' }}>
                게임 기록이 없습니다
              </div>
            ) : (
              <div style={{
                background: T.surface, border: `1px solid ${T.surfaceBorder}`,
                borderRadius: '4px', marginBottom: '40px', overflow: 'hidden',
              }}>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr style={{ borderBottom: `1px solid ${T.surfaceBorder}` }}>
                      {['날짜', '역할', '결과', '생존', '라운드', '시간'].map((h) => (
                        <th key={h} style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.1em', padding: '10px 16px', textAlign: 'left', fontWeight: 400 }}>{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {games.map((g) => (
                      <tr key={g.game_id} style={{ borderBottom: `1px solid ${T.surfaceBorder}` }}>
                        <td style={{ padding: '12px 16px', fontFamily: MONO, fontSize: '11px', color: T.textMuted }}>{formatDate(g.played_at)}</td>
                        <td style={{ padding: '12px 16px', fontFamily: SANS, fontSize: '13px', color: T.text }}>{ROLE_KO[g.role] ?? g.role}</td>
                        <td style={{ padding: '12px 16px', fontFamily: MONO, fontSize: '12px', color: g.won ? T.accent : T.danger }}>{g.won ? '승' : '패'}</td>
                        <td style={{ padding: '12px 16px', fontFamily: MONO, fontSize: '12px', color: T.textMuted }}>{g.survived ? 'O' : 'X'}</td>
                        <td style={{ padding: '12px 16px', fontFamily: MONO, fontSize: '12px', color: T.textMuted }}>{g.round_count}R</td>
                        <td style={{ padding: '12px 16px', fontFamily: MONO, fontSize: '12px', color: T.textMuted }}>{formatDuration(g.duration_sec)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        )}

        {/* ── Bottom buttons ─────────────────────────────────────────────── */}
        <div style={{ display: 'flex', gap: '12px' }}>
          <button
            onClick={() => navigate('/lobby')}
            style={{
              fontFamily: SANS, fontSize: '13px', fontWeight: 500,
              color: T.textMuted, background: 'transparent',
              border: `1px solid ${T.surfaceBorder}`, borderRadius: '2px',
              padding: '10px 20px', cursor: 'pointer', transition: 'all 150ms ease',
            }}
          >
            로비로 돌아가기
          </button>
          <button
            onClick={handleSignOut}
            style={{
              fontFamily: SANS, fontSize: '13px', fontWeight: 500,
              color: T.danger, background: T.dangerDim,
              border: `1px solid rgba(140,31,31,0.3)`, borderRadius: '2px',
              padding: '10px 20px', cursor: 'pointer', transition: 'all 150ms ease',
            }}
          >
            로그아웃
          </button>
        </div>

      </main>
    </div>
  )
}
```

- [ ] **Step 2: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/pages/ProfilePage.tsx
git commit -m "feat: add ProfilePage with nickname edit, stats cards, role breakdown, and recent games"
```

---

## Self-Review

**Spec coverage check:**
- ✅ Section 1 (닉네임 1:1 매핑, 방 입장 시 자동): Task 1 GetOrCreate no-overwrite + Task 3 resolvePlayerFull + Task 8 닉네임 입력 제거
- ✅ GET /api/me with display_name: Task 3 me() handler
- ✅ PUT /api/me: Task 3 updateMe()
- ✅ GET /api/me/stats: Task 2 + Task 3
- ✅ GET /api/me/games: Task 2 + Task 3
- ✅ authStore displayName: Task 6
- ✅ LobbyPage 프로필 버튼: Task 8 Step 7
- ✅ /profile 라우트: Task 9
- ✅ ProfilePage (닉네임 수정, 통계 카드, 역할별 테이블, 최근 게임): Task 10
- ✅ 로그아웃 버튼: Task 10 handleSignOut

**Type consistency check:**
- `updateMe` returns `{ player_id, display_name }` — matches handler response ✅
- `MyStatsResponse.by_role` is `Record<string, RoleStats>` — matches backend `byRole` map ✅
- `MyGameRecord` fields match backend `fiber.Map` keys ✅
- `resolvePlayerFull` returns `(playerID, displayName, error)` — used in createRoom/joinRoom/joinByCode ✅
