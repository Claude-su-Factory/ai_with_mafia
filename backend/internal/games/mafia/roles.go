package mafia

import (
	"math/rand"

	"ai-playground/internal/domain/entity"
)

const TotalPlayers = 6

// roleDist defines fixed role distribution for 6 players.
var roleDist = []entity.Role{
	entity.RoleMafia,
	entity.RoleMafia,
	entity.RolePolice,
	entity.RoleCitizen,
	entity.RoleCitizen,
	entity.RoleCitizen,
}

// AssignRoles shuffles and assigns roles to players in-place.
func AssignRoles(players []*entity.Player) {
	roles := make([]entity.Role, len(roleDist))
	copy(roles, roleDist)
	rand.Shuffle(len(roles), func(i, j int) { roles[i], roles[j] = roles[j], roles[i] })
	for i, p := range players {
		p.Role = roles[i]
	}
}

// MafiaIDs returns the IDs of mafia players.
func MafiaIDs(players []*entity.Player) []string {
	var ids []string
	for _, p := range players {
		if p.Role == entity.RoleMafia {
			ids = append(ids, p.ID)
		}
	}
	return ids
}
