package dto

type CreateRoomRequest struct {
	Name       string `json:"name"`
	MaxHumans  int    `json:"max_humans"`
	Visibility string `json:"visibility"` // "public" | "private"
}

type JoinRoomRequest struct {
	PlayerName string `json:"player_name"`
}

type JoinByCodeRequest struct {
	Code       string `json:"code"`
	PlayerName string `json:"player_name"`
}

type PlayerDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IsAlive bool   `json:"is_alive"`
	IsAI    bool   `json:"is_ai"`
}

type RoomResponse struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Visibility string      `json:"visibility"`
	JoinCode   string      `json:"join_code,omitempty"`
	HostID     string      `json:"host_id"`
	MaxHumans  int         `json:"max_humans"`
	Players    []PlayerDTO `json:"players"`
	Status     string      `json:"status"`
}

type JoinRoomResponse struct {
	RoomResponse
	PlayerID string `json:"player_id"`
}
