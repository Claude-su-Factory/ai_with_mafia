package repository

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AIHistoryRepository struct {
	db *pgxpool.Pool
}

func NewAIHistoryRepository(db *pgxpool.Pool) *AIHistoryRepository {
	return &AIHistoryRepository{db: db}
}

func (r *AIHistoryRepository) Save(ctx context.Context, roomID, playerID string, history []anthropic.MessageParam) error {
	b, err := json.Marshal(history)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO ai_histories (room_id, player_id, history_json, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (room_id, player_id) DO UPDATE
		SET history_json=$3, updated_at=NOW()
	`, roomID, playerID, b)
	return err
}

func (r *AIHistoryRepository) GetByRoom(ctx context.Context, roomID string) (map[string][]anthropic.MessageParam, error) {
	rows, err := r.db.Query(ctx, `
		SELECT player_id, history_json FROM ai_histories WHERE room_id=$1
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]anthropic.MessageParam)
	for rows.Next() {
		var playerID string
		var historyJSON []byte
		if err := rows.Scan(&playerID, &historyJSON); err != nil {
			return nil, err
		}
		var history []anthropic.MessageParam
		if err := json.Unmarshal(historyJSON, &history); err != nil {
			return nil, err
		}
		result[playerID] = history
	}
	return result, nil
}

func (r *AIHistoryRepository) DeleteByRoom(ctx context.Context, roomID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM ai_histories WHERE room_id=$1`, roomID)
	return err
}
