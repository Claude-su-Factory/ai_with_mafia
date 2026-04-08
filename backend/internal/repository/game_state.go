package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"ai-playground/internal/domain/entity"
)

// SavedGameState is the DB representation of a game state checkpoint.
type SavedGameState struct {
	RoomID     string
	Phase      string
	Round      int
	Players    []SavedPlayer
	NightKills map[string]string
}

// SavedPlayer is the per-player data stored in the checkpoint.
type SavedPlayer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	IsAlive bool   `json:"is_alive"`
	IsAI    bool   `json:"is_ai"`
}

type GameStateRepository struct {
	db *pgxpool.Pool
}

func NewGameStateRepository(db *pgxpool.Pool) *GameStateRepository {
	return &GameStateRepository{db: db}
}

func (r *GameStateRepository) Save(ctx context.Context, state SavedGameState) error {
	playersJSON, err := json.Marshal(state.Players)
	if err != nil {
		return err
	}
	nightKillsJSON, err := json.Marshal(state.NightKills)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO game_states (room_id, phase, round, players_json, night_kills, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (room_id) DO UPDATE
		SET phase=$2, round=$3, players_json=$4, night_kills=$5, updated_at=NOW()
	`, state.RoomID, state.Phase, state.Round, playersJSON, nightKillsJSON)
	return err
}

func (r *GameStateRepository) GetByRoomID(ctx context.Context, roomID string) (*SavedGameState, error) {
	row := r.db.QueryRow(ctx, `
		SELECT room_id, phase, round, players_json, night_kills
		FROM game_states WHERE room_id=$1
	`, roomID)

	var state SavedGameState
	var playersJSON, nightKillsJSON []byte
	if err := row.Scan(&state.RoomID, &state.Phase, &state.Round, &playersJSON, &nightKillsJSON); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(playersJSON, &state.Players); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(nightKillsJSON, &state.NightKills); err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *GameStateRepository) ListAll(ctx context.Context) ([]SavedGameState, error) {
	rows, err := r.db.Query(ctx, `
		SELECT room_id, phase, round, players_json, night_kills FROM game_states
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []SavedGameState
	for rows.Next() {
		var state SavedGameState
		var playersJSON, nightKillsJSON []byte
		if err := rows.Scan(&state.RoomID, &state.Phase, &state.Round, &playersJSON, &nightKillsJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(playersJSON, &state.Players); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(nightKillsJSON, &state.NightKills); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func (r *GameStateRepository) Delete(ctx context.Context, roomID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM game_states WHERE room_id=$1`, roomID)
	return err
}

// ToSavedPlayers converts entity players to SavedPlayer for checkpointing.
func ToSavedPlayers(players []*entity.Player) []SavedPlayer {
	result := make([]SavedPlayer, len(players))
	for i, p := range players {
		result[i] = SavedPlayer{
			ID:      p.ID,
			Name:    p.Name,
			Role:    string(p.Role),
			IsAlive: p.IsAlive,
			IsAI:    p.IsAI,
		}
	}
	return result
}
