package ws

import (
	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
)

// buildInitialStateRoomPayload returns the "room" object embedded in the
// initial_state WS event. It is the single source of truth for room shape on
// the WS path and must stay aligned with dto.RoomResponse (used on the HTTP
// path) so the frontend's Room type matches either transport.
//
// Notable policy carried here (mirrors ToRoomResponse in the platform package):
//   - AI players are excluded from the players list.
//   - join_code is always present as a string; it carries the real code only
//     for private rooms, otherwise empty — the key must exist so the frontend
//     contract can rely on it.
func buildInitialStateRoomPayload(room *entity.Room) map[string]any {
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

	joinCode := ""
	if room.Visibility == entity.VisibilityPrivate {
		joinCode = room.JoinCode
	}

	return map[string]any{
		"id":         room.ID,
		"name":       room.Name,
		"status":     string(room.GetStatus()),
		"host_id":    room.HostID,
		"visibility": string(room.Visibility),
		"join_code":  joinCode,
		"max_humans": room.MaxHumans,
		"players":    players,
	}
}
