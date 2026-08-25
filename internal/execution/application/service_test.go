package application

import (
	"context"
	"errors"
	"testing"
	"time"

	executiondomain "github.com/example/edge-task-orchestrator/internal/execution/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
)

type conflictExecutionRepository struct {
	item *executiondomain.Execution
}

func (*conflictExecutionRepository) Save(context.Context, *executiondomain.Execution) error {
	return nil
}
func (r *conflictExecutionRepository) Get(context.Context, string) (*executiondomain.Execution, error) {
	copy := *r.item
	return &copy, nil
}
func (*conflictExecutionRepository) Update(context.Context, *executiondomain.Execution) error {
	return executiondomain.ErrConflict
}
func (*conflictExecutionRepository) List(context.Context, executiondomain.Filter) ([]*executiondomain.Execution, error) {
	return nil, nil
}
func (*conflictExecutionRepository) ListRunnable(context.Context, time.Time, int) ([]*executiondomain.Execution, error) {
	return nil, nil
}
func (*conflictExecutionRepository) ListActive(context.Context) ([]*executiondomain.Execution, error) {
	return nil, nil
}

func TestReceiptUpdateConflictIsReturned(t *testing.T) {
	repository := &conflictExecutionRepository{item: &executiondomain.Execution{ID: "task-1", NodeID: "node-a", Status: executiondomain.StatusDispatched, Version: 1}}
	service := NewService(repository, nil, nil, clock.Fixed{Time: time.Now()})
	item, err := service.Receipt(context.Background(), "task-1", "node-a", ReceiptCommand{ReceiptID: "receipt-1", State: "running"})
	if item != nil || !errors.Is(err, executiondomain.ErrConflict) {
		t.Fatalf("conflict result: item=%v err=%v", item, err)
	}
}
