package application

import (
	"context"
	"errors"
	"testing"
	"time"

	executiondomain "github.com/example/edge-task-orchestrator/internal/execution/domain"
	executioninfrastructure "github.com/example/edge-task-orchestrator/internal/execution/infrastructure"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
)

func scoringExecution(id string) *executiondomain.Execution {
	return &executiondomain.Execution{ID: id, NodeID: "node-a", Status: executiondomain.StatusDispatched, Labels: map[string]string{"zone": "a"}, Version: 1}
}

func TestExecutionQueryReturnsOwnedSnapshot(t *testing.T) {
	repository := executioninfrastructure.NewMemoryRepository()
	ctx := context.Background()
	if err := repository.Save(ctx, scoringExecution("task-1")); err != nil {
		t.Fatal(err)
	}
	first, _ := repository.Get(ctx, "task-1")
	first.Priority = 77
	first.Labels["zone"] = "b"
	stored, _ := repository.Get(ctx, "task-1")
	if stored.Priority != 0 || stored.Labels["zone"] != "a" {
		t.Fatalf("stored execution was mutated: %+v", stored)
	}
}

func TestMissingExecutionUpdateCannotResurrectRecord(t *testing.T) {
	repository := executioninfrastructure.NewMemoryRepository()
	ctx := context.Background()
	err := repository.Update(ctx, scoringExecution("ghost"))
	if !errors.Is(err, executiondomain.ErrNotFound) {
		t.Fatalf("missing update error = %v", err)
	}
	if _, err := repository.Get(ctx, "ghost"); !errors.Is(err, executiondomain.ErrNotFound) {
		t.Fatalf("missing execution became visible: %v", err)
	}
}

func TestExecutionUpdateRejectsVersionJump(t *testing.T) {
	repository := executioninfrastructure.NewMemoryRepository()
	ctx := context.Background()
	if err := repository.Save(ctx, scoringExecution("task-2")); err != nil {
		t.Fatal(err)
	}
	item, _ := repository.Get(ctx, "task-2")
	item.Version += 2
	if err := repository.Update(ctx, item); !errors.Is(err, executiondomain.ErrConflict) {
		t.Fatalf("version jump error = %v", err)
	}
}

func TestReceiptConflictDoesNotReturnSuccess(t *testing.T) {
	repository := &conflictExecutionRepository{item: scoringExecution("task-3")}
	service := NewService(repository, nil, nil, clock.Fixed{Time: time.Now()})
	item, err := service.Receipt(context.Background(), "task-3", "node-a", ReceiptCommand{ReceiptID: "receipt-3", State: "running"})
	if item != nil || !errors.Is(err, executiondomain.ErrConflict) {
		t.Fatalf("conflict result: item=%v err=%v", item, err)
	}
}
