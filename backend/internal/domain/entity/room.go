package entity

import "sync"

type RoomStatus string
type Visibility string

const (
	RoomStatusWaiting  RoomStatus = "waiting"
	RoomStatusPlaying  RoomStatus = "playing"
	RoomStatusFinished RoomStatus = "finished"

	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

type Room struct {
	mu         sync.RWMutex
	ID         string
	Name       string
	Visibility Visibility
	JoinCode   string // 비공개 방 6자리 코드
	HostID     string
	MaxHumans  int
	Players    []*Player
	Status     RoomStatus
}

func (r *Room) AddPlayer(p *Player) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Players = append(r.Players, p)
}

func (r *Room) RemovePlayer(playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.Players {
		if p.ID == playerID {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			return
		}
	}
}

func (r *Room) HumanCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, p := range r.Players {
		if !p.IsAI {
			count++
		}
	}
	return count
}

func (r *Room) PlayerByID(id string) *Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (r *Room) SetStatus(s RoomStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Status = s
}

func (r *Room) GetStatus() RoomStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Status
}

func (r *Room) GetPlayers() []*Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Player, len(r.Players))
	copy(result, r.Players)
	return result
}
