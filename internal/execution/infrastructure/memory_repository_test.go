package infrastructure

import (
	"context"
	"testing"

	executiondomain "github.com/example/edge-task-orchestrator/internal/execution/domain"
)

func executionFixture(id string) *executiondomain.Execution {
	return &executiondomain.Execution{ID: id, Status: executiondomain.StatusQueued, Labels: map[string]string{"region": "north"}, Version: 1}
}

func TestMemoryRepositoryOwnershipAndVersioning(t *testing.T) {
	ctx := context.Background()

	t.Run("query snapshot", func(t *testing.T) {
		repository := NewMemoryRepository()
		if err := repository.Save(ctx, executionFixture("task-1")); err != nil {
			t.Fatal(err)
		}
		first, _ := repository.Get(ctx, "task-1")
		first.Priority = 99
		first.Labels["region"] = "south"
		stored, _ := repository.Get(ctx, "task-1")
		if stored.Priority != 0 || stored.Labels["region"] != "north" {
			t.Fatalf("query result changed stored execution: %+v", stored)
		}
	})

	t.Run("missing update", func(t *testing.T) {
		repository := NewMemoryRepository()
		if err := repository.Update(ctx, executionFixture("ghost")); err == nil {
			t.Fatal("missing execution was inserted by update")
		}
		if _, err := repository.Get(ctx, "ghost"); err == nil {
			t.Fatal("missing execution became queryable")
		}
	})

	t.Run("version jump", func(t *testing.T) {
		repository := NewMemoryRepository()
		if err := repository.Save(ctx, executionFixture("task-2")); err != nil {
			t.Fatal(err)
		}
		item, _ := repository.Get(ctx, "task-2")
		item.Version += 2
		if err := repository.Update(ctx, item); err == nil {
			t.Fatal("non-consecutive version was accepted")
		}
	})
}
