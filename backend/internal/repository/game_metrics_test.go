package repository

import (
	"context"
	"testing"
	"time"
)

func TestGameMetricsRepo_NilPool_AllMethodsNoOp(t *testing.T) {
	repo := NewGameMetricsRepository(nil)
	ctx := context.Background()

	if err := repo.Create(ctx, GameMetricInit{GameID: "g1", RoomID: "r1", StartedAt: time.Now()}); err != nil {
		t.Errorf("Create nil pool: %v, want nil", err)
	}
	if err := repo.Finalize(ctx, GameMetricFinal{GameID: "g1", EndedAt: time.Now()}); err != nil {
		t.Errorf("Finalize nil pool: %v, want nil", err)
	}
	if err := repo.AddAIUsage(ctx, "g1", AIUsage{TokensIn: 10}); err != nil {
		t.Errorf("AddAIUsage nil pool: %v, want nil", err)
	}
	if err := repo.IncrementAdImpression(ctx, "waiting", "g1"); err != nil {
		t.Errorf("IncrementAdImpression nil pool: %v, want nil", err)
	}
	if err := repo.RecordQuickMatch(ctx, "g1", "joined", 123); err != nil {
		t.Errorf("RecordQuickMatch nil pool: %v, want nil", err)
	}
}
