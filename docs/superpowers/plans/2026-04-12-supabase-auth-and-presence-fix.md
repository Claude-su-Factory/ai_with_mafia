# Supabase Auth + Presence Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Google 소셜 로그인으로 Google 계정 = 고정 player_id를 구현하고, 중복 세션 방지 및 뒤로가기 즉각 반영 버그를 수정한다.

**Architecture:** Supabase는 인증 계층만 담당(HS256 JWT). 백엔드는 JWT를 검증해 `auth_id → player_id` 매핑을 `users` 테이블에 영구 저장한다. 활성 세션은 Redis에 `user_session:{player_id} → room_id`로 추적하고, 뒤로가기는 `pagehide` + `navigator.sendBeacon`으로 즉각 처리한다.

**Tech Stack:** Go, Fiber, PostgreSQL, Redis, React, TypeScript, Zustand, @supabase/supabase-js, github.com/golang-jwt/jwt/v5

---

## File Map

| 파일 | 변경 | 책임 |
|------|------|------|
| `backend/config/config.go` | 수정 | SupabaseConfig 추가 |
| `backend/migrations/000006_create_users.up.sql` | 신규 | users 테이블 생성 |
| `backend/migrations/000006_create_users.down.sql` | 신규 | users 테이블 삭제 |
| `backend/internal/repository/user.go` | 신규 | UserRepository: GetOrCreate, GetByAuthID |
| `backend/internal/repository/session.go` | 신규 | SessionRepository: Set/Get/Delete (Redis) |
| `backend/internal/platform/auth.go` | 신규 | ValidateJWT 함수 (HS256) |
| `backend/internal/platform/auth_test.go` | 신규 | ValidateJWT 단위 테스트 |
| `backend/internal/platform/ws/hub.go` | 수정 | ForceRemove 메서드 추가 |
| `backend/internal/platform/handler.go` | 수정 | 인증 통합, /api/me, /leave 엔드포인트 |
| `backend/internal/platform/handler_test.go` | 수정 | mockHub에 ForceRemove 추가 |
| `backend/cmd/server/main.go` | 수정 | UserRepo/SessionRepo 주입, WS ?token= 파라미터 |
| `frontend/src/lib/supabase.ts` | 신규 | Supabase 클라이언트 초기화 |
| `frontend/src/store/authStore.ts` | 신규 | 인증 상태 (user, playerID, signInWithGoogle, signOut) |
| `frontend/src/main.tsx` | 수정 | AppInit 컴포넌트로 authStore 초기화 |
| `frontend/src/pages/LandingPage.tsx` | 수정 | 로그인 상태 확인 + Google 로그인 버튼 |
| `frontend/src/api.ts` | 수정 | Authorization: Bearer 헤더, 409 처리 |
| `frontend/src/store/gameStore.ts` | 수정 | WS URL에 token 파라미터, player_id를 authStore에서 |
| `frontend/src/pages/LobbyPage.tsx` | 수정 | localStorage.setItem 제거, 409 다이얼로그 |
| `frontend/src/pages/RoomPage.tsx` | 수정 | localStorage 체크 제거, pagehide sendBeacon |
| `frontend/.env.development` | 수정 | VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY |
| `frontend/.env.production` | 수정 | VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY |

---

## Task 1: 백엔드 — JWT 의존성 + SupabaseConfig

**Files:**
- Modify: `backend/config/config.go`
- Modify: `backend/go.mod` (via go get)

- [ ] **Step 1: JWT 라이브러리 설치**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go get github.com/golang-jwt/jwt/v5
```

Expected: `go: added github.com/golang-jwt/jwt/v5 ...`

- [ ] **Step 2: SupabaseConfig를 config.go에 추가**

`backend/config/config.go`에서 `Config` struct와 새 타입을 추가:

```go
type Config struct {
	Server   ServerConfig    `toml:"server"`
	Database DatabaseConfig  `toml:"database"`
	Redis    RedisConfig     `toml:"redis"`
	AI       AIConfig        `toml:"ai"`
	Game     GameConfig      `toml:"game"`
	Personas []PersonaConfig `toml:"personas"`
	Supabase SupabaseConfig  `toml:"supabase"`
}

type SupabaseConfig struct {
	JWTSecret string `toml:"jwt_secret"`
}
```

- [ ] **Step 3: config.toml에 supabase 섹션 추가**

`backend/config.toml` 파일 끝에 추가 (파일이 존재하는지 먼저 확인):

```toml
[supabase]
jwt_secret = ""   # Supabase 프로젝트 설정 > API > JWT Secret
```

- [ ] **Step 4: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/config/config.go backend/go.mod backend/go.sum
git commit -m "feat: add SupabaseConfig and jwt/v5 dependency"
```

---

## Task 2: 백엔드 — DB 마이그레이션 (users 테이블)

**Files:**
- Create: `backend/migrations/000006_create_users.up.sql`
- Create: `backend/migrations/000006_create_users.down.sql`

- [ ] **Step 1: up 마이그레이션 생성**

```sql
-- backend/migrations/000006_create_users.up.sql
CREATE TABLE users (
    auth_id      TEXT PRIMARY KEY,
    player_id    TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- [ ] **Step 2: down 마이그레이션 생성**

```sql
-- backend/migrations/000006_create_users.down.sql
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/migrations/000006_create_users.up.sql backend/migrations/000006_create_users.down.sql
git commit -m "feat: add users table migration for auth_id → player_id mapping"
```

---

## Task 3: 백엔드 — UserRepository

**Files:**
- Create: `backend/internal/repository/user.go`

- [ ] **Step 1: user.go 생성**

```go
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// GetOrCreate returns the player_id for auth_id. On first call it creates a new
// user with a fresh UUID. On subsequent calls it updates display_name and returns
// the existing player_id (PostgreSQL UPSERT returns the existing row's player_id).
func (r *UserRepository) GetOrCreate(ctx context.Context, authID, displayName string) (string, error) {
	playerID := uuid.NewString()
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (auth_id, player_id, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (auth_id) DO UPDATE SET display_name = EXCLUDED.display_name
		RETURNING player_id
	`, authID, playerID, displayName).Scan(&playerID)
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
git commit -m "feat: add UserRepository with GetOrCreate and GetByAuthID"
```

---

## Task 4: 백엔드 — SessionRepository (Redis)

**Files:**
- Create: `backend/internal/repository/session.go`

- [ ] **Step 1: session.go 생성**

```go
package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const sessionTTL = 24 * time.Hour

type SessionRepository struct {
	rdb *redis.Client
}

func NewSessionRepository(rdb *redis.Client) *SessionRepository {
	return &SessionRepository{rdb: rdb}
}

// Set stores player → room mapping with 24-hour TTL.
func (r *SessionRepository) Set(ctx context.Context, playerID, roomID string) error {
	return r.rdb.Set(ctx, "user_session:"+playerID, roomID, sessionTTL).Err()
}

// Get returns the room_id for a player. Returns empty string (no error) if not found.
func (r *SessionRepository) Get(ctx context.Context, playerID string) (string, error) {
	roomID, err := r.rdb.Get(ctx, "user_session:"+playerID).Result()
	if err == redis.Nil {
		return "", nil
	}
	return roomID, err
}

// Delete removes the session entry for a player.
func (r *SessionRepository) Delete(ctx context.Context, playerID string) error {
	return r.rdb.Del(ctx, "user_session:"+playerID).Err()
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
git add backend/internal/repository/session.go
git commit -m "feat: add SessionRepository for active room tracking in Redis"
```

---

## Task 5: 백엔드 — ValidateJWT + 단위 테스트

**Files:**
- Create: `backend/internal/platform/auth.go`
- Create: `backend/internal/platform/auth_test.go`

- [ ] **Step 1: 실패 테스트 작성**

`backend/internal/platform/auth_test.go`:

```go
package platform_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"ai-playground/internal/platform"
)

func makeToken(t *testing.T, secret, sub, fullName string, expired bool) string {
	t.Helper()
	exp := time.Now().Add(time.Hour)
	if expired {
		exp = time.Now().Add(-time.Hour)
	}
	claims := jwt.MapClaims{
		"sub": sub,
		"exp": exp.Unix(),
		"user_metadata": map[string]any{"full_name": fullName},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	return tok
}

func TestValidateJWT_Valid(t *testing.T) {
	tok := makeToken(t, "secret", "auth-uuid", "Alice", false)
	authID, name, err := platform.ValidateJWT(tok, "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authID != "auth-uuid" {
		t.Errorf("authID = %q, want %q", authID, "auth-uuid")
	}
	if name != "Alice" {
		t.Errorf("displayName = %q, want %q", name, "Alice")
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	tok := makeToken(t, "secret", "id", "Bob", false)
	_, _, err := platform.ValidateJWT(tok, "wrong")
	if err == nil {
		t.Error("expected error for wrong secret, got nil")
	}
}

func TestValidateJWT_Expired(t *testing.T) {
	tok := makeToken(t, "secret", "id", "Carol", true)
	_, _, err := platform.ValidateJWT(tok, "secret")
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestValidateJWT_Empty(t *testing.T) {
	_, _, err := platform.ValidateJWT("", "secret")
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}
}
```

- [ ] **Step 2: 테스트 실패 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./internal/platform/ -run TestValidateJWT -v
```

Expected: `cannot find package` 또는 컴파일 에러 (auth.go 없음)

- [ ] **Step 3: auth.go 구현**

`backend/internal/platform/auth.go`:

```go
package platform

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type supabaseClaims struct {
	jwt.RegisteredClaims
	UserMetadata struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	} `json:"user_metadata"`
}

// ValidateJWT validates a Supabase HS256 JWT and returns (authID, displayName, error).
// authID is the Supabase user UUID (JWT "sub" claim).
// displayName is taken from user_metadata.full_name, falling back to user_metadata.name.
func ValidateJWT(tokenStr, secret string) (authID, displayName string, err error) {
	if tokenStr == "" {
		return "", "", errors.New("empty token")
	}
	token, err := jwt.ParseWithClaims(tokenStr, &supabaseClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", "", err
	}
	claims, ok := token.Claims.(*supabaseClaims)
	if !ok || !token.Valid {
		return "", "", errors.New("invalid token claims")
	}
	authID = claims.Subject
	displayName = claims.UserMetadata.FullName
	if displayName == "" {
		displayName = claims.UserMetadata.Name
	}
	if displayName == "" && len(authID) >= 8 {
		displayName = authID[:8]
	}
	return authID, displayName, nil
}
```

- [ ] **Step 4: 테스트 통과 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./internal/platform/ -run TestValidateJWT -v
```

Expected: 4개 테스트 모두 PASS

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/platform/auth.go backend/internal/platform/auth_test.go
git commit -m "feat: add ValidateJWT for Supabase HS256 token validation"
```

---

## Task 6: 백엔드 — Hub ForceRemove + GameHub 인터페이스

**Files:**
- Modify: `backend/internal/platform/ws/hub.go`
- Modify: `backend/internal/platform/handler.go` (GameHub 인터페이스만)

- [ ] **Step 1: GameHub 인터페이스에 ForceRemove 추가**

`backend/internal/platform/handler.go`에서 `GameHub` 인터페이스를 찾아 수정:

```go
// GameHub is implemented by ws.Hub; defined here to avoid circular imports.
type GameHub interface {
	StartGame(roomID string) error
	RestartGame(roomID string) error
	ForceRemove(playerID, roomID string)
}
```

- [ ] **Step 2: hub.go에 ForceRemove 구현**

`backend/internal/platform/ws/hub.go`의 `RestartGame` 메서드 바로 뒤에 추가:

```go
// ForceRemove immediately removes a player without waiting for the grace timer.
// Used by the leave-beacon endpoint when a player intentionally navigates away.
func (h *Hub) ForceRemove(playerID, roomID string) {
	// Cancel grace timer if there is one pending for this player.
	h.pdMu.Lock()
	if pd, ok := h.pendingDisconnects[playerID]; ok {
		pd.timer.Stop()
		delete(h.pendingDisconnects, playerID)
	}
	h.pdMu.Unlock()

	// Remove from local client map if still connected.
	h.mu.Lock()
	if clients, ok := h.rooms[roomID]; ok {
		delete(clients, playerID)
		if len(clients) == 0 {
			delete(h.rooms, roomID)
		}
	}
	h.mu.Unlock()

	h.doRemove(roomID, playerID)
}
```

- [ ] **Step 3: handler_test.go의 mockHub에 ForceRemove 추가**

`backend/internal/platform/handler_test.go`에서 `mockHub` struct를 찾아 메서드 추가:

```go
func (m *mockHub) ForceRemove(_, _ string) {}
```

- [ ] **Step 4: 빌드 + 테스트 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./... && go test ./...
```

Expected: 빌드 성공, 테스트 모두 PASS

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/platform/ws/hub.go backend/internal/platform/handler.go backend/internal/platform/handler_test.go
git commit -m "feat: add Hub.ForceRemove for immediate player removal on page unload"
```

---

## Task 7: 백엔드 — Handler 전면 수정 (인증 + 세션 + leave + /me)

**Files:**
- Modify: `backend/internal/platform/handler.go`

Handler struct에 `userRepo`, `sessionRepo`, `jwtSecret` 추가. 모든 write 엔드포인트에서 JWT로 player_id를 가져오도록 변경. `/api/me`와 `/api/rooms/:id/leave` 추가.

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

// checkActiveSession returns a 409 response if the player is already in a live room.
// Cleans up stale Redis entries if the room no longer exists.
func (h *Handler) checkActiveSession(c *fiber.Ctx, playerID string) (stop bool) {
	existingRoomID, err := h.sessionRepo.Get(c.Context(), playerID)
	if err != nil || existingRoomID == "" {
		return false
	}
	if _, err := h.rooms.GetByID(existingRoomID); err == nil {
		c.Status(fiber.StatusConflict).JSON(fiber.Map{ //nolint
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
	if stop := h.checkActiveSession(c, playerID); stop {
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
	_ = h.sessionRepo.Set(c.Context(), playerID, room.ID)
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
	if stop := h.checkActiveSession(c, playerID); stop {
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
	_ = h.sessionRepo.Set(c.Context(), playerID, room.ID)
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
	if stop := h.checkActiveSession(c, playerID); stop {
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
	_ = h.sessionRepo.Set(c.Context(), playerID, room.ID)
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
// No JWT auth — uses player_id from request body (sendBeacon cannot set headers).
func (h *Handler) leaveRoom(c *fiber.Ctx) error {
	var req leaveRequest
	if err := c.BodyParser(&req); err != nil || req.PlayerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "player_id required"})
	}
	roomID := c.Params("id")
	h.gameHub.ForceRemove(req.PlayerID, roomID)
	_ = h.sessionRepo.Delete(c.Context(), req.PlayerID)
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 2: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음. `NewHandler` 시그니처가 바뀌었으므로 `main.go` 에러가 날 수 있음 — Task 8에서 수정.

- [ ] **Step 3: handler_test.go setupApp 수정**

`backend/internal/platform/handler_test.go`의 `setupApp` 함수에서 `NewHandler` 호출을 수정:

```go
func setupApp(t *testing.T) (*fiber.App, *RoomService) {
	t.Helper()
	svc := NewRoomService(nil, zap.NewNop())
	h := NewHandler(svc, &mockHub{}, nil, nil, "")
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	h.RegisterRoutes(app)
	return app, svc
}
```

Note: userRepo=nil, sessionRepo=nil, jwtSecret="" — 기존 테스트는 인증이 필요 없는 `listRooms`/`getRoom`만 테스트하므로 nil 허용.

- [ ] **Step 4: 빌드 + 테스트 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./... && go test ./...
```

Expected: 빌드 성공, 테스트 모두 PASS (main.go 에러는 아직 있을 수 있음)

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/internal/platform/handler.go backend/internal/platform/handler_test.go
git commit -m "feat: refactor Handler to use JWT auth, session tracking, and leave endpoint"
```

---

## Task 8: 백엔드 — main.go 배선

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: main.go 수정**

`backend/cmd/server/main.go`에서 다음 변경을 적용:

**1) import에 `context` 확인** (이미 있음)

**2) Repositories 섹션 수정** — userRepo, sessionRepo 추가:

```go
// --- Repositories ---
gameStateRepo := repository.NewGameStateRepository(pool)
aiHistoryRepo := repository.NewAIHistoryRepository(pool)
gameResultRepo := repository.NewGameResultRepository(pool)
userRepo := repository.NewUserRepository(pool)
sessionRepo := repository.NewSessionRepository(rdb)
```

**3) HTTP routes 섹션 수정** — NewHandler에 새 인수 전달:

```go
handler := platform.NewHandler(roomSvc, gameHub, userRepo, sessionRepo, cfg.Supabase.JWTSecret)
handler.RegisterRoutes(app)
```

**4) WS 핸들러 수정** — `?player_id=` → `?token=` + JWT 검증:

```go
app.Get("/ws/rooms/:id", fiberws.New(func(c *fiberws.Conn) {
	roomID := c.Params("id")
	tokenStr := c.Query("token")
	authID, _, err := platform.ValidateJWT(tokenStr, cfg.Supabase.JWTSecret)
	if err != nil {
		logger.Warn("ws: invalid token", zap.String("room_id", roomID), zap.Error(err))
		_ = c.Close()
		return
	}
	playerID, err := userRepo.GetByAuthID(context.Background(), authID)
	if err != nil || playerID == "" {
		logger.Warn("ws: user not found for auth_id", zap.String("auth_id", authID))
		_ = c.Close()
		return
	}
	gameHub.ServeWS(c, roomID, playerID)
}))
```

- [ ] **Step 2: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go build ./...
```

Expected: 에러 없음

- [ ] **Step 3: 전체 테스트 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && go test ./...
```

Expected: 모든 테스트 PASS

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add backend/cmd/server/main.go
git commit -m "feat: wire UserRepo/SessionRepo into server, switch WS to ?token= param"
```

---

## Task 9: 프론트엔드 — Supabase 설치 + 클라이언트 + authStore + main.tsx

**Files:**
- Create: `frontend/src/lib/supabase.ts`
- Create: `frontend/src/store/authStore.ts`
- Modify: `frontend/src/main.tsx`

- [ ] **Step 1: @supabase/supabase-js 설치**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm install @supabase/supabase-js
```

Expected: `added ... packages`

- [ ] **Step 2: supabase.ts 생성**

```typescript
// frontend/src/lib/supabase.ts
import { createClient } from '@supabase/supabase-js'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL as string
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY as string

export const supabase = createClient(supabaseUrl, supabaseAnonKey)
```

- [ ] **Step 3: authStore.ts 생성**

```typescript
// frontend/src/store/authStore.ts
import { create } from 'zustand'
import type { User } from '@supabase/supabase-js'
import { supabase } from '../lib/supabase'

interface AuthStore {
  user: User | null
  playerID: string
  loading: boolean
  initialize: () => Promise<void>
  signInWithGoogle: () => Promise<void>
  signOut: () => Promise<void>
  getAccessToken: () => Promise<string>
}

// Guard against double-initialization in React StrictMode.
let initialized = false

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  playerID: '',
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
        const data = await res.json() as { player_id: string }
        set({ user: session.user, playerID: data.player_id, loading: false })
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
          const data = await res.json() as { player_id: string }
          set({ user: session.user, playerID: data.player_id })
        } else {
          set({ user: session.user })
        }
      } else if (event === 'SIGNED_OUT') {
        set({ user: null, playerID: '' })
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
    set({ user: null, playerID: '' })
  },

  async getAccessToken() {
    const { data: { session } } = await supabase.auth.getSession()
    return session?.access_token ?? ''
  },
}))
```

- [ ] **Step 4: main.tsx에 AppInit 추가**

`frontend/src/main.tsx`를 다음으로 교체:

```tsx
import { StrictMode, useEffect, type ReactNode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import './index.css'
import LandingPage from './pages/LandingPage'
import LobbyPage from './pages/LobbyPage'
import RoomPage from './pages/RoomPage'
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
])

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppInit>
      <RouterProvider router={router} />
    </AppInit>
  </StrictMode>,
)
```

- [ ] **Step 5: 빌드 확인 (VITE_SUPABASE_URL이 없어서 경고 가능 — 에러 아님)**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 빌드 성공 (env 변수 경고는 Task 15에서 설정)

- [ ] **Step 6: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/lib/supabase.ts frontend/src/store/authStore.ts frontend/src/main.tsx frontend/package.json frontend/package-lock.json
git commit -m "feat: add Supabase client, authStore, and AppInit for auth initialization"
```

---

## Task 10: 프론트엔드 — LandingPage 인증 게이트

**Files:**
- Modify: `frontend/src/pages/LandingPage.tsx`

LandingPage의 모든 `navigate('/lobby')` 버튼을 Google 로그인 흐름으로 교체. 이미 로그인된 경우 `/lobby`로 자동 리디렉트.

- [ ] **Step 1: LandingPage.tsx 상단 import 수정**

파일 첫 줄 `import { useEffect, useRef, useState, useCallback } from 'react'`에 이미 있는 것들. 여기에 `useAuthStore` import 추가:

```typescript
import { useAuthStore } from '../store/authStore'
```

- [ ] **Step 2: 컴포넌트 상단에 인증 로직 추가**

`export default function LandingPage()` 내부, `const navigate = useNavigate()` 바로 아래에 추가:

```typescript
const { user, loading, signInWithGoogle } = useAuthStore()

useEffect(() => {
  if (!loading && user) {
    navigate('/lobby')
  }
}, [user, loading, navigate])

function handleCTA() {
  if (user) {
    navigate('/lobby')
  } else {
    void signInWithGoogle()
  }
}
```

- [ ] **Step 3: 모든 navigate('/lobby') onClick을 handleCTA로 교체**

파일 내 `onClick={() => navigate('/lobby')}` 패턴 4곳을 모두 `onClick={handleCTA}`로 교체:

```bash
grep -n "onClick={() => navigate('/lobby')}" /Users/yuhojin/Desktop/ai_side/frontend/src/pages/LandingPage.tsx
```

해당 줄의 `onClick={() => navigate('/lobby')}` 전부를 `onClick={handleCTA}`로 교체.

- [ ] **Step 4: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/pages/LandingPage.tsx
git commit -m "feat: add Google login gate to LandingPage, auto-redirect if already logged in"
```

---

## Task 11: 프론트엔드 — api.ts Authorization 헤더

**Files:**
- Modify: `frontend/src/api.ts`

- [ ] **Step 1: api.ts 전체 교체**

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
  player_name: string
  max_humans?: number
}

export interface JoinRoomParams {
  room_id: string
  player_name: string
}

export interface JoinByCodeParams {
  code: string
  player_name: string
}

export interface JoinResponse {
  player_id: string
  id: string
}

export function listRooms() {
  return request<import('./types').Room[]>('/rooms')
}

export function createRoom(params: CreateRoomParams) {
  return request<JoinResponse>('/rooms', {
    method: 'POST',
    headers: { 'X-Player-Name': params.player_name },
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
    body: JSON.stringify({ player_name: params.player_name }),
  })
}

export function joinByCode(params: JoinByCodeParams) {
  return request<JoinResponse>('/rooms/join/code', {
    method: 'POST',
    body: JSON.stringify(params),
  })
}

export function startGame(roomID: string) {
  return request<void>(`/rooms/${roomID}/start`, { method: 'POST' })
}

export function restartGame(roomID: string) {
  return request<void>(`/rooms/${roomID}/restart`, { method: 'POST' })
}
```

Note: `startGame`과 `restartGame`에서 `playerID` 파라미터 제거 — 이제 Authorization 헤더로 처리됨.

- [ ] **Step 2: 빌드 확인 (startGame/restartGame 시그니처 변경으로 다른 파일 에러 날 수 있음)**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build 2>&1 | head -30
```

에러가 있으면 Task 12 이후에서 해당 호출부도 수정할 것임. 지금은 에러 목록만 확인.

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/api.ts
git commit -m "feat: add Authorization header to API requests, attach roomID to 409 errors"
```

---

## Task 12: 프론트엔드 — gameStore.ts WS token 파라미터

**Files:**
- Modify: `frontend/src/store/gameStore.ts`

- [ ] **Step 1: gameStore.ts의 connect 함수 수정**

`frontend/src/store/gameStore.ts`에서 `connect(roomID: string)` 함수를 찾아 수정.

현재 코드 (lines 159-172 근처):
```typescript
connect(roomID: string) {
    currentRoomID = roomID
    const playerID = localStorage.getItem(`player_id_${roomID}`) ?? ''
    set({ playerID, wsStatus: 'connecting' })
    // ...
    const url = `${protocol}//${window.location.host}/ws/rooms/${roomID}?player_id=${playerID}`
```

교체할 코드:
```typescript
connect(roomID: string) {
    currentRoomID = roomID
    set({ wsStatus: 'connecting' })

    void (async () => {
      const { useAuthStore } = await import('./authStore')
      const { playerID, getAccessToken } = useAuthStore.getState()
      const token = await getAccessToken()
      set({ playerID })

      clearReconnectTimer()
      if (ws) {
        ws.onclose = null
        ws.close()
      }

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const url = `${protocol}//${window.location.host}/ws/rooms/${roomID}?token=${token}`
      ws = new WebSocket(url)
```

그 뒤로 `ws.onopen`, `ws.onmessage`, `ws.onclose` 등은 변경 없음. async IIFE 블록을 닫는 `})()`를 connect 함수 끝에 추가.

- [ ] **Step 2: startGame/restartGame 호출부 수정**

`gameStore.ts`에서 `startGame`, `restartGame` 호출 시 playerID 파라미터를 제거:

파일에서 `startGame(roomID!, playerID)` → `startGame(roomID!)`
파일에서 `restartGame(roomID!, playerID)` → `restartGame(roomID!)`

`ResultOverlay.tsx`에서도 같은 패턴을 확인하고 수정:

```bash
grep -rn "startGame\|restartGame" /Users/yuhojin/Desktop/ai_side/frontend/src/
```

playerID 인수를 받는 모든 호출에서 playerID 제거.

- [ ] **Step 3: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/store/gameStore.ts
git commit -m "feat: use Supabase JWT token in WebSocket URL, get playerID from authStore"
```

---

## Task 13: 프론트엔드 — LobbyPage localStorage 제거 + 409 처리

**Files:**
- Modify: `frontend/src/pages/LobbyPage.tsx`

- [ ] **Step 1: handleJoinRoom 수정**

`frontend/src/pages/LobbyPage.tsx`의 `handleJoinRoom` 함수를 찾아 수정.

현재:
```typescript
const res = await joinRoom({ room_id: joiningRoom.id, player_name: joinName.trim() })
localStorage.setItem(`player_id_${joiningRoom.id}`, res.player_id)
navigate(`/rooms/${joiningRoom.id}`)
```

교체:
```typescript
const res = await joinRoom({ room_id: joiningRoom.id, player_name: joinName.trim() })
navigate(`/rooms/${res.id}`)
```

catch 블록에 409 처리 추가:
```typescript
} catch (e: unknown) {
  if (e instanceof Error && (e as Error & { roomID?: string }).roomID) {
    const existingRoomID = (e as Error & { roomID: string }).roomID
    if (confirm('이미 입장한 방이 있습니다. 돌아가시겠습니까?')) {
      navigate(`/rooms/${existingRoomID}`)
    }
    return
  }
  setJoinError(e instanceof Error ? e.message : '참가에 실패했습니다.')
}
```

- [ ] **Step 2: handleCreateRoom 수정**

`handleCreateRoom`에서도 같은 패턴 적용:

현재:
```typescript
const res = await createRoom({ ... })
localStorage.setItem(`player_id_${res.id}`, res.player_id)
navigate(`/rooms/${res.id}`)
```

교체:
```typescript
const res = await createRoom({ ... })
navigate(`/rooms/${res.id}`)
```

catch 블록에 409 처리:
```typescript
} catch (e: unknown) {
  if (e instanceof Error && (e as Error & { roomID?: string }).roomID) {
    const existingRoomID = (e as Error & { roomID: string }).roomID
    if (confirm('이미 입장한 방이 있습니다. 돌아가시겠습니까?')) {
      navigate(`/rooms/${existingRoomID}`)
    }
    return
  }
  setCreateError(e instanceof Error ? e.message : '방 생성에 실패했습니다.')
}
```

- [ ] **Step 3: handleJoinByCode 수정**

`handleJoinByCode`에도 동일 패턴:

현재:
```typescript
const res = await joinByCode({ code: codeInput.trim(), player_name: codePlayerName.trim() })
localStorage.setItem(`player_id_${res.id}`, res.player_id)
navigate(`/rooms/${res.id}`)
```

교체:
```typescript
const res = await joinByCode({ code: codeInput.trim(), player_name: codePlayerName.trim() })
navigate(`/rooms/${res.id}`)
```

catch 블록:
```typescript
} catch (e: unknown) {
  if (e instanceof Error && (e as Error & { roomID?: string }).roomID) {
    const existingRoomID = (e as Error & { roomID: string }).roomID
    if (confirm('이미 입장한 방이 있습니다. 돌아가시겠습니까?')) {
      navigate(`/rooms/${existingRoomID}`)
    }
    return
  }
  setCodeError(e instanceof Error ? e.message : '코드 참가에 실패했습니다.')
}
```

- [ ] **Step 4: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/pages/LobbyPage.tsx
git commit -m "feat: remove localStorage player_id, handle 409 active-session redirect in LobbyPage"
```

---

## Task 14: 프론트엔드 — RoomPage pagehide + localStorage 제거

**Files:**
- Modify: `frontend/src/pages/RoomPage.tsx`

- [ ] **Step 1: localStorage 체크 제거 + pagehide 핸들러 추가**

`frontend/src/pages/RoomPage.tsx`에서 두 가지를 수정.

현재 localStorage 체크 (lines 32-43 근처):
```typescript
useEffect(() => {
    if (!roomID) return
    const playerID = localStorage.getItem(`player_id_${roomID}`)
    if (!playerID) {
      navigate('/lobby')
      return
    }
    connect(roomID)
    return () => {
      disconnect()
    }
  }, [roomID])
```

교체 (localStorage 체크 제거 + pagehide 추가):
```typescript
const { playerID } = useAuthStore()

useEffect(() => {
  if (!roomID) return
  if (!playerID) {
    navigate('/lobby')
    return
  }
  connect(roomID)
  return () => {
    disconnect()
  }
}, [roomID, playerID])

useEffect(() => {
  if (!roomID || !playerID) return
  const handlePageHide = () => {
    navigator.sendBeacon(
      `/api/rooms/${roomID}/leave`,
      new Blob([JSON.stringify({ player_id: playerID })], { type: 'application/json' }),
    )
  }
  window.addEventListener('pagehide', handlePageHide)
  return () => window.removeEventListener('pagehide', handlePageHide)
}, [roomID, playerID])
```

파일 상단 import에 `useAuthStore` 추가:
```typescript
import { useAuthStore } from '../store/authStore'
```

컴포넌트 내부에서 `useAuthStore()` 호출 추가 (connect, disconnect 구조분해 근처):
```typescript
const { playerID } = useAuthStore()
```

- [ ] **Step 2: ResultOverlay.tsx에서 localStorage 제거**

`frontend/src/components/ResultOverlay.tsx`의 `handleLeave` 함수 확인:

```bash
grep -n "localStorage" /Users/yuhojin/Desktop/ai_side/frontend/src/components/ResultOverlay.tsx
```

`localStorage.removeItem(`player_id_${roomID}`)` 줄을 삭제. 세션 해제는 `/leave` sendBeacon이 처리함.

- [ ] **Step 3: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/pages/RoomPage.tsx frontend/src/components/ResultOverlay.tsx
git commit -m "feat: add pagehide sendBeacon for immediate leave, remove localStorage player_id checks"
```

---

## Task 15: 프론트엔드 — 환경 변수 파일

**Files:**
- Modify: `frontend/.env.development`
- Modify: `frontend/.env.production`

- [ ] **Step 1: .env.development 수정**

```
# Development: AdBanner renders null when these are empty
VITE_ADSENSE_CLIENT=
VITE_ADSENSE_SLOT_WAITING=
VITE_ADSENSE_SLOT_RESULT=

# Supabase — fill in from Supabase project settings > API
VITE_SUPABASE_URL=https://your-project.supabase.co
VITE_SUPABASE_ANON_KEY=your-anon-key
```

- [ ] **Step 2: .env.production 수정**

`frontend/.env.production`을 읽어 기존 내용 확인 후 Supabase 키 섹션 추가:

```
# Supabase — fill in real values before deploy
VITE_SUPABASE_URL=https://your-project.supabase.co
VITE_SUPABASE_ANON_KEY=your-anon-key
```

- [ ] **Step 3: 최종 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/.env.development frontend/.env.production
git commit -m "feat: add Supabase env variable placeholders"
```

---

## 최종 검증 체크리스트

- [ ] Supabase 프로젝트에서 Google OAuth 공급자 활성화
- [ ] Supabase 프로젝트 → Authentication → URL Configuration → Redirect URLs에 `http://localhost:5173/lobby` 추가
- [ ] `backend/config.toml`에 실제 JWT Secret 입력 (Supabase 프로젝트 설정 → API → JWT Secret)
- [ ] `.env.development`에 실제 VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY 입력
- [ ] 서버 + 프론트 실행 후 Google 로그인 흐름 수동 확인
- [ ] 같은 Google 계정으로 두 탭에서 방 입장 시도 → 두 번째 탭에서 409 다이얼로그 표시 확인
- [ ] 대기실에서 뒤로가기 → 방에서 즉각 제거 확인 (sendBeacon 작동)
- [ ] 새로고침 → grace timer로 재접속 허용 확인 (sendBeacon 미발송)
