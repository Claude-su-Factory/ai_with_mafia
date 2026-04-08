package entity

type Role string

const (
	RoleMafia   Role = "mafia"
	RoleCitizen Role = "citizen"
	RolePolice  Role = "police"
)

type Player struct {
	ID      string
	Name    string
	Role    Role
	IsAlive bool
	IsAI    bool
}

func NewPlayer(id, name string, isAI bool) *Player {
	return &Player{
		ID:      id,
		Name:    name,
		IsAlive: true,
		IsAI:    isAI,
	}
}
