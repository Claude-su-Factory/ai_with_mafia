package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
)

type Client struct {
	PlayerID string
	RoomID   string
	Role     entity.Role
	conn     *websocket.Conn
	send     chan []byte
}

type pendingDisconnect struct {
	timer  *time.Timer
	roomID string
	role   entity.Role
}

type Hub struct {
	mu          sync.RWMutex
	rooms       map[string]map[string]*Client // roomID -> playerID -> client
	roomService RoomService
	gameManager GameManager
	logger      *zap.Logger
	serverCtx   context.Context
	rdb         *redis.Client
	instanceID  string
	graceSec    int

	pdMu               sync.Mutex
	pendingDisconnects map[string]*pendingDisconnect // playerID -> pending
}

type RoomService interface {
	GetByID(id string) (*entity.Room, error)
	RemovePlayer(roomID, playerID string) *entity.Room
}

type GameManager interface {
	StartGame(ctx context.Context, room *entity.Room) error
	RestartGame(ctx context.Context, room *entity.Room) error
	NotifyEvent(roomID string, event entity.GameEvent)
	TryRecover(ctx context.Context, roomID string)
	GetSnapshot(roomID string) *dto.GameSnapshot
	StopGame(roomID string)
	ReplaceWithAI(roomID, playerID string, role entity.Role) error
}

func NewHub(ctx context.Context, rooms RoomService, games GameManager, logger *zap.Logger, rdb *redis.Client, instanceID string, graceSec int) *Hub {
	return &Hub{
		rooms:              make(map[string]map[string]*Client),
		roomService:        rooms,
		gameManager:        games,
		logger:             logger,
		serverCtx:          ctx,
		rdb:                rdb,
		instanceID:         instanceID,
		graceSec:           graceSec,
		pendingDisconnects: make(map[string]*pendingDisconnect),
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[c.RoomID] == nil {
		h.rooms[c.RoomID] = make(map[string]*Client)
	}
	h.rooms[c.RoomID][c.PlayerID] = c
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if clients, ok := h.rooms[c.RoomID]; ok {
		delete(clients, c.PlayerID)
		if len(clients) == 0 {
			delete(h.rooms, c.RoomID)
		}
	}
	h.mu.Unlock()

	if h.graceSec > 0 {
		h.startGraceTimer(c)
	} else {
		h.doRemove(c.RoomID, c.PlayerID)
	}
}

// startGraceTimer starts a reconnect grace period timer for the disconnecting client.
func (h *Hub) startGraceTimer(c *Client) {
	pd := &pendingDisconnect{
		roomID: c.RoomID,
		role:   c.Role,
	}
	pd.timer = time.AfterFunc(time.Duration(h.graceSec)*time.Second, func() {
		h.pdMu.Lock()
		delete(h.pendingDisconnects, c.PlayerID)
		h.pdMu.Unlock()
		h.doRemove(c.RoomID, c.PlayerID)
	})

	h.pdMu.Lock()
	h.pendingDisconnects[c.PlayerID] = pd
	h.pdMu.Unlock()
}

// doRemove removes the player from the room service and notifies via Pub/Sub.
// If the game is in progress, the leaving player is replaced with an AI agent.
// If all human players have left, the game is stopped and the room is cleaned up.
func (h *Hub) doRemove(roomID, playerID string) {
	// Capture the player's role and game status BEFORE removal
	// (role is needed for AI replacement, status to know if game is active).
	var playerRole entity.Role
	var playerName string
	var wasPlaying bool
	if preRoom, err := h.roomService.GetByID(roomID); err == nil && preRoom != nil {
		if p := preRoom.PlayerByID(playerID); p != nil {
			playerRole = p.Role
			playerName = p.Name
		}
		wasPlaying = preRoom.GetStatus() == entity.RoomStatusPlaying
	}

	room := h.roomService.RemovePlayer(roomID, playerID)

	// Notify other instances
	if h.rdb != nil {
		p := pubsubPayload{
			Origin:    h.instanceID,
			EventType: "player_removed",
			PlayerID:  playerID,
		}
		if err := publishToRoom(h.serverCtx, h.rdb, roomID, p); err != nil {
			h.logger.Warn("redis publish player_removed failed",
				zap.String("room_id", roomID),
				zap.String("player_id", playerID),
				zap.Error(err))
		}
	}

	if room == nil {
		// All humans left — stop game and broadcast game over.
		h.gameManager.StopGame(roomID)
		h.Broadcast(roomID, dto.GameEventDTO{
			Type:    string(entity.EventGameOver),
			Payload: map[string]any{"reason": "all_humans_left"},
		}, false)
		return
	}

	// If game is in progress, replace the leaving player with an AI agent.
	if wasPlaying && playerRole != "" {
		h.Broadcast(roomID, dto.GameEventDTO{
			Type: string(entity.EventPlayerReplaced),
			Payload: map[string]any{
				"player_id": playerID,
				"message":   "플레이어가 이탈하여 AI로 대체됩니다.",
			},
		}, false)

		go func() {
			if err := h.gameManager.ReplaceWithAI(roomID, playerID, playerRole); err != nil {
				h.logger.Error("ReplaceWithAI failed",
					zap.String("room_id", roomID),
					zap.String("player_id", playerID),
					zap.Error(err))
			}
		}()
		return
	}

	// 대기실 퇴장 — player_left broadcast
	h.Broadcast(roomID, dto.GameEventDTO{
		Type: string(entity.EventPlayerLeft),
		Payload: map[string]any{
			"player_id":   playerID,
			"player_name": playerName,
		},
	}, false)
}

// Broadcast sends a message to all local clients in a room and relays via Redis Pub/Sub.
// If mafiaOnly is true, sends only to mafia role clients.
func (h *Hub) Broadcast(roomID string, payload any, mafiaOnly bool) {
	b, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("broadcast: json marshal failed", zap.String("room_id", roomID), zap.Error(err))
		return
	}

	h.broadcastLocal(roomID, b, mafiaOnly)

	// Relay to other instances via Redis
	if h.rdb != nil {
		p := pubsubPayload{
			Origin:    h.instanceID,
			MafiaOnly: mafiaOnly,
			Data:      b,
		}
		if err := publishToRoom(h.serverCtx, h.rdb, roomID, p); err != nil {
			h.logger.Warn("redis publish failed", zap.String("room_id", roomID), zap.Error(err))
		}
	}
}

// broadcastLocal delivers a pre-marshalled message to local clients only.
func (h *Hub) broadcastLocal(roomID string, data []byte, mafiaOnly bool) {
	h.mu.RLock()
	clients := h.rooms[roomID]
	h.mu.RUnlock()

	for _, c := range clients {
		if mafiaOnly && c.Role != entity.RoleMafia {
			continue
		}
		select {
		case c.send <- data:
		default:
			h.logger.Warn("client send channel full, dropping message",
				zap.String("player_id", c.PlayerID),
				zap.String("room_id", roomID))
		}
	}
}

// StartSubscriber subscribes to all room channels and relays messages from other instances.
func (h *Hub) StartSubscriber(ctx context.Context) {
	h.startSubscriber(ctx)
}

func (h *Hub) startSubscriber(ctx context.Context) {
	if h.rdb == nil {
		return
	}
	sub := h.rdb.PSubscribe(ctx, "room:*")
	go func() {
		defer sub.Close()
		ch := sub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				h.handlePubSubMessage(msg)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (h *Hub) handlePubSubMessage(msg *redis.Message) {
	var p pubsubPayload
	if err := json.Unmarshal([]byte(msg.Payload), &p); err != nil {
		h.logger.Warn("pubsub: invalid payload", zap.Error(err))
		return
	}
	// Skip messages from self
	if p.Origin == h.instanceID {
		return
	}

	// Extract roomID from channel name "room:{id}"
	if len(msg.Channel) <= 5 {
		return
	}
	roomID := msg.Channel[5:]

	switch p.EventType {
	case "player_reconnected":
		// Cancel grace timer on this instance if the player reconnected elsewhere
		h.pdMu.Lock()
		if pd, ok := h.pendingDisconnects[p.PlayerID]; ok {
			pd.timer.Stop()
			delete(h.pendingDisconnects, p.PlayerID)
		}
		h.pdMu.Unlock()
	case "player_removed":
		// Idempotent — already handled locally or not relevant
	default:
		// WS relay message
		h.broadcastLocal(roomID, p.Data, p.MafiaOnly)
	}
}

// StartGame implements platform.GameHub
func (h *Hub) StartGame(roomID string) error {
	room, err := h.roomService.GetByID(roomID)
	if err != nil {
		return err
	}
	return h.gameManager.StartGame(h.serverCtx, room)
}

// RestartGame implements platform.GameHub
func (h *Hub) RestartGame(roomID string) error {
	room, err := h.roomService.GetByID(roomID)
	if err != nil {
		return err
	}
	return h.gameManager.RestartGame(h.serverCtx, room)
}

// SendToPlayer sends a message to a single player in a room.
// If the player is not connected locally or the send channel is full, a warning is logged.
func (h *Hub) SendToPlayer(roomID, playerID string, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		h.logger.Warn("SendToPlayer: marshal failed",
			zap.String("player_id", playerID),
			zap.String("room_id", roomID),
			zap.Error(err))
		return
	}

	h.mu.RLock()
	var c *Client
	if clients, ok := h.rooms[roomID]; ok {
		c = clients[playerID]
	}
	h.mu.RUnlock()

	if c == nil {
		return
	}
	select {
	case c.send <- b:
	default:
		h.logger.Warn("SendToPlayer: send channel full, dropping message",
			zap.String("player_id", playerID),
			zap.String("room_id", roomID))
	}
}

// UpdateClientRole updates the role of a connected client (called after role assignment).
func (h *Hub) UpdateClientRole(roomID, playerID string, role entity.Role) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.rooms[roomID]; ok {
		if c, ok := clients[playerID]; ok {
			c.Role = role
		}
	}
}

// ServeWS handles a WebSocket connection for a room.
func (h *Hub) ServeWS(c *websocket.Conn, roomID, playerID string) {
	// Trigger recovery if this is a playing room with no active leader
	h.gameManager.TryRecover(h.serverCtx, roomID)

	// Cancel grace timer if this is a reconnecting player
	h.pdMu.Lock()
	pd, wasPending := h.pendingDisconnects[playerID]
	if wasPending {
		pd.timer.Stop()
		delete(h.pendingDisconnects, playerID)
	}
	h.pdMu.Unlock()

	if wasPending {
		// Publish reconnect event to other instances
		if h.rdb != nil {
			p := pubsubPayload{
				Origin:    h.instanceID,
				EventType: "player_reconnected",
				PlayerID:  playerID,
			}
			if err := publishToRoom(h.serverCtx, h.rdb, roomID, p); err != nil {
				h.logger.Warn("redis publish player_reconnected failed",
					zap.String("room_id", roomID),
					zap.String("player_id", playerID),
					zap.Error(err))
			}
		}
	}

	room, err := h.roomService.GetByID(roomID)
	if err != nil {
		h.logger.Warn("ws: room not found, closing",
			zap.String("room_id", roomID),
			zap.String("player_id", playerID))
		_ = c.Close()
		return
	}

	player := room.PlayerByID(playerID)
	if player == nil {
		h.logger.Warn("ws: player not in room, closing",
			zap.String("room_id", roomID),
			zap.String("player_id", playerID))
		_ = c.Close()
		return
	}

	role := player.Role
	// Restore role from pending disconnect if available (role may not be assigned yet at room level)
	if wasPending && pd.role != "" {
		role = pd.role
	}

	client := &Client{
		PlayerID: playerID,
		RoomID:   roomID,
		Role:     role,
		conn:     c,
		send:     make(chan []byte, 256),
	}
	h.Register(client)
	defer h.Unregister(client)

	// Build initial_state payload
	snapshot := h.gameManager.GetSnapshot(roomID)

	players := room.GetPlayers()
	playerList := make([]map[string]any, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, map[string]any{
			"id":       p.ID,
			"name":     p.Name,
			"is_alive": p.IsAlive,
			"is_ai":    p.IsAI,
		})
	}

	roomPayload := map[string]any{
		"id":         room.ID,
		"name":       room.Name,
		"status":     string(room.Status),
		"host_id":    room.HostID,
		"visibility": string(room.Visibility),
		"join_code":  room.JoinCode,
		"players":    playerList,
	}

	var gamePayload any
	if snapshot != nil {
		gamePayload = map[string]any{
			"phase":               snapshot.Phase,
			"round":               snapshot.Round,
			"timer_remaining_sec": snapshot.TimerRemainingSec,
			"alive_player_ids":    snapshot.AlivePlayerIDs,
			"votes":               snapshot.Votes,
		}
	}

	initialStateMsg, err := json.Marshal(map[string]any{
		"type": "initial_state",
		"payload": map[string]any{
			"room":    roomPayload,
			"game":    gamePayload,
			"my_role": string(role),
		},
	})
	if err != nil {
		h.logger.Warn("ws initial_state marshal failed",
			zap.String("player_id", playerID),
			zap.String("room_id", roomID),
			zap.Error(err))
		return
	}
	if err := c.WriteMessage(websocket.TextMessage, initialStateMsg); err != nil {
		h.logger.Warn("ws initial_state send failed",
			zap.String("player_id", playerID),
			zap.String("room_id", roomID),
			zap.Error(err))
		return
	}

	// 같은 방 모든 클라이언트에게 입장 알림 (자기 자신 포함 — 프론트엔드에서 필터링)
	h.Broadcast(roomID, dto.GameEventDTO{
		Type: string(entity.EventPlayerJoined),
		Payload: map[string]any{
			"player_id":   playerID,
			"player_name": player.Name,
		},
	}, false)

	// 쓰기 goroutine
	ctx, cancel := context.WithCancel(h.serverCtx)
	defer cancel()

	go func() {
		for {
			select {
			case msg, ok := <-client.send:
				if !ok {
					return
				}
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					h.logger.Warn("ws write failed",
						zap.String("player_id", playerID),
						zap.String("room_id", roomID),
						zap.Error(err))
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 읽기 루프 — per-player rate limit (500 ms min interval)
	const msgRateLimit = 200 * time.Millisecond
	var lastMsg time.Time

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}

		if now := time.Now(); now.Sub(lastMsg) < msgRateLimit {
			h.logger.Warn("rate limit: dropping message",
				zap.String("player_id", playerID),
				zap.String("room_id", roomID))
			continue
		} else {
			lastMsg = now
		}

		var action dto.ActionRequest
		if err := json.Unmarshal(msg, &action); err != nil {
			h.logger.Warn("invalid action", zap.Error(err))
			continue
		}

		h.gameManager.NotifyEvent(roomID, entity.GameEvent{
			Type:     entity.GameEventType(action.Type),
			PlayerID: playerID,
			Payload:  map[string]any{"raw": action},
		})
	}
}
