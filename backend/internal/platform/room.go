package platform

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
	"ai-playground/internal/repository"
)

const (
	codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLen   = 6
)

type RoomService struct {
	mu       sync.RWMutex
	rooms    map[string]*entity.Room
	db       *pgxpool.Pool
	roomRepo *repository.RoomRepository
	logger   *zap.Logger
}

func NewRoomService(pool *pgxpool.Pool, logger *zap.Logger) *RoomService {
	return &RoomService{
		rooms:    make(map[string]*entity.Room),
		db:       pool,
		roomRepo: repository.NewRoomRepository(pool),
		logger:   logger,
	}
}

func (s *RoomService) Create(req dto.CreateRoomRequest, hostID, hostName string) (*entity.Room, error) {
	if req.MaxHumans < 1 || req.MaxHumans > 6 {
		return nil, fmt.Errorf("max_humans must be between 1 and 6")
	}

	id, err := newID()
	if err != nil {
		return nil, fmt.Errorf("generate room id: %w", err)
	}
	vis := entity.Visibility(req.Visibility)
	if vis != entity.VisibilityPublic && vis != entity.VisibilityPrivate {
		vis = entity.VisibilityPublic
	}

	var joinCode string
	if vis == entity.VisibilityPrivate {
		joinCode, err = generateCode()
		if err != nil {
			return nil, fmt.Errorf("generate join code: %w", err)
		}
	}

	host := entity.NewPlayer(hostID, hostName, false)
	room := &entity.Room{
		ID:         id,
		Name:       req.Name,
		Visibility: vis,
		JoinCode:   joinCode,
		HostID:     hostID,
		MaxHumans:  req.MaxHumans,
		Status:     entity.RoomStatusWaiting,
	}
	room.AddPlayer(host)

	s.mu.Lock()
	s.rooms[id] = room
	s.mu.Unlock()

	// Persist to DB (non-fatal)
	if s.db != nil {
		if err := s.roomRepo.Save(context.Background(), room); err != nil {
			s.logger.Error("failed to persist room to db", zap.String("room_id", id), zap.Error(err))
		}
	}

	return room, nil
}

func (s *RoomService) Join(roomID, playerID, playerName string) (*entity.Room, error) {
	room, err := s.GetByID(roomID)
	if err != nil {
		return nil, fmt.Errorf("room not found")
	}
	if room.HumanCount() >= room.MaxHumans {
		return nil, fmt.Errorf("room is full")
	}
	p := entity.NewPlayer(playerID, playerName, false)
	room.AddPlayer(p)
	return room, nil
}

func (s *RoomService) JoinByCode(code, playerID, playerName string) (*entity.Room, error) {
	s.mu.RLock()
	var roomID string
	for _, room := range s.rooms {
		if strings.EqualFold(room.JoinCode, code) {
			roomID = room.ID
			break
		}
	}
	s.mu.RUnlock()

	// DB fallback if not in memory
	if roomID == "" && s.db != nil {
		rows, err := s.db.Query(context.Background(), `
			SELECT id FROM rooms WHERE UPPER(join_code)=UPPER($1) AND status != 'finished'
		`, code)
		if err == nil {
			defer rows.Close()
			if rows.Next() {
				_ = rows.Scan(&roomID)
			}
		}
	}

	if roomID == "" {
		return nil, fmt.Errorf("invalid join code")
	}
	return s.Join(roomID, playerID, playerName)
}

func (s *RoomService) GetByID(id string) (*entity.Room, error) {
	s.mu.RLock()
	room, ok := s.rooms[id]
	s.mu.RUnlock()
	if ok {
		return room, nil
	}

	// DB fallback
	if s.db == nil {
		return nil, fmt.Errorf("room not found")
	}
	dbRoom, err := s.roomRepo.GetByID(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("room not found")
	}

	// Load into memory cache
	s.mu.Lock()
	s.rooms[id] = dbRoom
	s.mu.Unlock()

	return dbRoom, nil
}

func (s *RoomService) ListPublic() []*entity.Room {
	if s.db != nil {
		dbRooms, err := s.roomRepo.ListPublic(context.Background())
		if err != nil {
			s.logger.Error("failed to list public rooms from db", zap.Error(err))
		} else {
			// Only return rooms that are active in memory (excludes stale DB records)
			s.mu.RLock()
			result := make([]*entity.Room, 0, len(dbRooms))
			for _, r := range dbRooms {
				if mem, ok := s.rooms[r.ID]; ok {
					result = append(result, mem)
				}
			}
			s.mu.RUnlock()
			return result
		}
	}
	// Fallback to memory
	s.mu.RLock()
	defer s.mu.RUnlock()
	var rooms []*entity.Room
	for _, r := range s.rooms {
		if r.Visibility == entity.VisibilityPublic {
			rooms = append(rooms, r)
		}
	}
	return rooms
}

// LoadRoom loads a room into the memory cache (used during recovery).
func (s *RoomService) LoadRoom(room *entity.Room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.ID] = room
}

// RemovePlayer removes a player, transfers host if needed, deletes room if empty.
// Returns the updated room (nil if deleted).
func (s *RoomService) RemovePlayer(roomID, playerID string) *entity.Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return nil
	}

	room.RemovePlayer(playerID)

	if room.HumanCount() == 0 {
		delete(s.rooms, roomID)
		if s.db != nil {
			_ = s.roomRepo.Delete(context.Background(), roomID)
		}
		return nil
	}

	// 방장이 나갔으면 다음 사람에게 이전
	if room.HostID == playerID {
		for _, p := range room.Players {
			if !p.IsAI {
				room.HostID = p.ID
				break
			}
		}
	}

	return room
}

func (s *RoomService) Delete(roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, roomID)
	if s.roomRepo != nil {
		_ = s.roomRepo.Delete(context.Background(), roomID)
	}
}

func ToRoomResponse(room *entity.Room) dto.RoomResponse {
	players := make([]dto.PlayerDTO, 0, len(room.Players))
	for _, p := range room.Players {
		if p.IsAI {
			continue
		}
		players = append(players, dto.PlayerDTO{
			ID:      p.ID,
			Name:    p.Name,
			IsAlive: p.IsAlive,
			IsAI:    p.IsAI,
		})
	}
	resp := dto.RoomResponse{
		ID:         room.ID,
		Name:       room.Name,
		Visibility: string(room.Visibility),
		HostID:     room.HostID,
		MaxHumans:  room.MaxHumans,
		Players:    players,
		Status:     string(room.GetStatus()),
	}
	if room.Visibility == entity.VisibilityPrivate {
		resp.JoinCode = room.JoinCode
	}
	return resp
}

func generateCode() (string, error) {
	b := make([]byte, codeLen)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		if err != nil {
			return "", err
		}
		b[i] = codeChars[n.Int64()]
	}
	return string(b), nil
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// RoomToJSON is a helper for WebSocket broadcasts.
func RoomToJSON(room *entity.Room) []byte {
	resp := ToRoomResponse(room)
	b, _ := json.Marshal(resp)
	return b
}
