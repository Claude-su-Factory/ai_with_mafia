package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"ai-playground/config"
	"ai-playground/internal/ai"
	"ai-playground/internal/domain/entity"
	"ai-playground/internal/platform"
	"ai-playground/internal/platform/ws"
	"ai-playground/internal/repository"
)

func main() {
	// --- Config ---
	cfgPath := "config.toml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// --- Logger ---
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer logger.Sync()

	// --- Instance UUID ---
	instanceID := uuid.New().String()
	logger.Info("instance started", zap.String("instance_id", instanceID))

	// --- DB Migrations ---
	migrationsPath := "migrations"
	if err := repository.RunMigrations(cfg.Database.DSN, migrationsPath); err != nil {
		logger.Fatal("db migration failed", zap.Error(err))
	}
	logger.Info("migrations applied")

	// --- DB Pool ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := repository.NewPool(ctx, cfg.Database.DSN)
	if err != nil {
		logger.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()
	logger.Info("database connected")

	// --- Redis ---
	rdb, err := repository.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		logger.Fatal("redis connect failed", zap.Error(err))
	}
	defer rdb.Close()
	logger.Info("redis connected")

	// --- Repositories ---
	gameStateRepo := repository.NewGameStateRepository(pool)
	aiHistoryRepo := repository.NewAIHistoryRepository(pool)
	gameResultRepo := repository.NewGameResultRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	sessionRepo := repository.NewSessionRepository(rdb)
	gameMetricsRepo := repository.NewGameMetricsRepository(pool)

	// --- Room Service ---
	roomSvc := platform.NewRoomService(pool, logger)

	// --- Leader Lock ---
	leaderLock := platform.NewLeaderLock(rdb)

	// --- AI ---
	if cfg.AI.APIKey == "" {
		logger.Fatal("ai.api_key is not set in config")
	}
	personaPool := ai.NewPersonaPool(cfg.Personas)
	aiManager := ai.NewManager(&cfg.AI, personaPool, cfg.AI.APIKey, logger, aiHistoryRepo, &aiMetricsAdapter{repo: gameMetricsRepo})

	// --- Game Manager ---
	gm := platform.NewGameManager(&cfg.Game.Mafia, aiManager, personaPool, leaderLock, instanceID, gameStateRepo, aiHistoryRepo, gameResultRepo, roomSvc, logger)

	// --- WebSocket Hub ---
	gameHub := ws.NewHub(ctx, roomSvc, gm, logger, rdb, instanceID, cfg.Server.ReconnectGraceSec)

	// Wire gameManager → Hub callbacks
	gm.GameEventFunc = func(roomID string, event entity.GameEvent) {
		msg := map[string]any{
			"type":    string(event.Type),
			"payload": event.Payload,
		}
		if event.PlayerID != "" {
			gameHub.SendToPlayer(roomID, event.PlayerID, msg)
		} else {
			gameHub.Broadcast(roomID, msg, event.MafiaOnly)
		}
	}
	gm.UpdateRoleFunc = func(roomID, playerID string, role entity.Role) {
		gameHub.UpdateClientRole(roomID, playerID, role)
		gameHub.SendToPlayer(roomID, playerID, map[string]any{
			"type":    "role_assigned",
			"payload": map[string]any{"role": string(role)},
		})
	}

	// Wire AI → Hub callbacks + AI → game callbacks
	aiManager.SetCallbacks(
		func(roomID, playerID, playerName, message string, mafiaOnly bool) {
			evType := entity.EventChat
			if mafiaOnly {
				evType = entity.EventMafiaChat
			}
			if gm.GameEventFunc != nil {
				gm.GameEventFunc(roomID, entity.GameEvent{
					Type:      evType,
					MafiaOnly: mafiaOnly,
					Payload: map[string]any{
						"sender_id":   playerID,
						"sender_name": playerName,
						"message":     message,
					},
				})
			}
		},
		func(roomID, playerID, targetID string) {
			if err := gm.DispatchAction(roomID, playerID, entity.Action{
				Type:    "vote",
				Payload: map[string]any{"target_id": targetID},
			}); err != nil {
				logger.Error("vote callback: HandleAction failed",
					zap.String("room_id", roomID),
					zap.String("player_id", playerID),
					zap.Error(err))
			}
		},
		func(roomID, playerID, actionType, targetID string) {
			if err := gm.DispatchAction(roomID, playerID, entity.Action{
				Type:    actionType,
				Payload: map[string]any{"target_id": targetID},
			}); err != nil {
				logger.Error("night callback: HandleAction failed",
					zap.String("room_id", roomID),
					zap.String("player_id", playerID),
					zap.String("action", actionType),
					zap.Error(err))
			}
		},
	)

	// --- Start Pub/Sub subscriber ---
	gameHub.StartSubscriber(ctx)

	// --- Recover orphan games ---
	recoverOrphanGames(ctx, gm, gameStateRepo, aiHistoryRepo, roomSvc, logger)

	// --- JWT Public Key (ES256 / Supabase JWK) ---
	xBytes, err := base64.RawURLEncoding.DecodeString(cfg.Supabase.JWTPublicKeyX)
	if err != nil {
		logger.Fatal("invalid jwt_public_key_x", zap.Error(err))
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(cfg.Supabase.JWTPublicKeyY)
	if err != nil {
		logger.Fatal("invalid jwt_public_key_y", zap.Error(err))
	}
	jwtPubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	// --- Fiber App ---
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	app.Use(recover.New())
	app.Use(cors.New())

	// HTTP routes
	handler := platform.NewHandler(roomSvc, gameHub, userRepo, sessionRepo, gameResultRepo, gameMetricsRepo, jwtPubKey)
	handler.RegisterRoutes(app)

	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/rooms/:id", fiberws.New(func(c *fiberws.Conn) {
		roomID := c.Params("id")
		tokenStr := c.Query("token")
		authID, _, err := platform.ValidateJWT(tokenStr, jwtPubKey)
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

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		logger.Info("server starting", zap.String("addr", addr))
		if err := app.Listen(addr); err != nil {
			logger.Error("server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down...")
	cancel()
	_ = app.Shutdown()
}

// recoverOrphanGames finds playing rooms with no leader and restarts their game loops.
func recoverOrphanGames(
	ctx context.Context,
	gm *platform.GameManager,
	gameStateRepo *repository.GameStateRepository,
	aiHistoryRepo *repository.AIHistoryRepository,
	roomSvc *platform.RoomService,
	logger *zap.Logger,
) {
	states, err := gameStateRepo.ListAll(ctx)
	if err != nil {
		logger.Error("recoverOrphanGames: list game states failed", zap.Error(err))
		return
	}

	for _, state := range states {
		room, err := roomSvc.GetByID(state.RoomID)
		if err != nil {
			// Room no longer exists — delete the orphan game_state to prevent repeated warnings on restart.
			if delErr := gameStateRepo.Delete(ctx, state.RoomID); delErr != nil {
				logger.Warn("recoverOrphanGames: failed to delete orphan state",
					zap.String("room_id", state.RoomID), zap.Error(delErr))
			} else {
				logger.Info("recoverOrphanGames: deleted orphan game state",
					zap.String("room_id", state.RoomID))
			}
			continue
		}

		// Rebuild players from checkpoint into room
		for _, sp := range state.Players {
			if room.PlayerByID(sp.ID) == nil {
				p := entity.NewPlayer(sp.ID, sp.Name, sp.IsAI)
				p.Role = entity.Role(sp.Role)
				p.IsAlive = sp.IsAlive
				room.AddPlayer(p)
			}
		}
		roomSvc.LoadRoom(room)

		// Load AI histories
		histories, err := aiHistoryRepo.GetByRoom(ctx, state.RoomID)
		if err != nil {
			logger.Warn("recoverOrphanGames: load ai histories failed",
				zap.String("room_id", state.RoomID), zap.Error(err))
			histories = nil
		}

		if err := gm.RecoverGame(ctx, room, histories); err != nil {
			logger.Error("recoverOrphanGames: recover failed",
				zap.String("room_id", state.RoomID), zap.Error(err))
		} else {
			logger.Info("game recovered", zap.String("room_id", state.RoomID))
		}
	}
}

// aiMetricsAdapter bridges ai.MetricsSink (takes ai.AIUsage) to the concrete
// repository.GameMetricsRepository (takes repository.AIUsage). The two structs
// are field-for-field identical so a direct value conversion is safe — keeping
// internal/ai free of a type-level dependency on the concrete repo type.
type aiMetricsAdapter struct {
	repo *repository.GameMetricsRepository
}

func (a *aiMetricsAdapter) AddAIUsage(ctx context.Context, gameID string, u ai.AIUsage) error {
	if a == nil || a.repo == nil {
		return nil
	}
	return a.repo.AddAIUsage(ctx, gameID, repository.AIUsage(u))
}
