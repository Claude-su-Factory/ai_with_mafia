package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// pubsubPayload is the envelope published to Redis room channels.
type pubsubPayload struct {
	Origin    string `json:"origin"`              // instanceID of sender
	MafiaOnly bool   `json:"mafia_only"`
	EventType string `json:"event_type,omitempty"` // internal: "player_reconnected", "player_removed", or empty for WS relay
	PlayerID  string `json:"player_id,omitempty"` // for reconnect/removed events
	Data      []byte `json:"data"`                // raw JSON message for WS relay
}

func roomChannel(roomID string) string {
	return fmt.Sprintf("room:%s", roomID)
}

func publishToRoom(ctx context.Context, rdb *redis.Client, roomID string, payload pubsubPayload) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return rdb.Publish(ctx, roomChannel(roomID), b).Err()
}
