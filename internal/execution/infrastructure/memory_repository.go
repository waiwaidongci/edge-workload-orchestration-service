package infrastructure

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	executiondomain "github.com/example/edge-task-orchestrator/internal/execution/domain"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*executiondomain.Execution
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*executiondomain.Execution)}
}

func (r *MemoryRepository) Save(ctx context.Context, item *executiondomain.Execution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; ok {
		return fmt.Errorf("execution %s already exists", item.ID)
	}
	r.items[item.ID] = clone(item)
	return nil
}
func (r *MemoryRepository) Get(ctx context.Context, executionID string) (*executiondomain.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[executionID]
	if !ok {
		return nil, executiondomain.ErrNotFound
	}
	return clone(item), nil
}
func (r *MemoryRepository) Update(ctx context.Context, item *executiondomain.Execution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.items[item.ID]
	if !ok {
		return executiondomain.ErrNotFound
	}
	if item.Version <= existing.Version {
		return fmt.Errorf("%w: stale execution version", executiondomain.ErrConflict)
	}
	r.items[item.ID] = clone(item)
	return nil
}

func (r *MemoryRepository) List(ctx context.Context, filter executiondomain.Filter) ([]*executiondomain.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*executiondomain.Execution, 0)
	for _, item := range r.items {
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.NodeID != "" && item.NodeID != filter.NodeID {
			continue
		}
		if filter.TemplateID != "" && item.TemplateID != filter.TemplateID {
			continue
		}
		result = append(result, clone(item))
	}
	sortItems(result)
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result, nil
}

func (r *MemoryRepository) ListRunnable(ctx context.Context, now time.Time, limit int) ([]*executiondomain.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*executiondomain.Execution, 0)
	for _, item := range r.items {
		if item.Runnable(now) {
			result = append(result, clone(item))
		}
	}
	sortItems(result)
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *MemoryRepository) ListActive(ctx context.Context) ([]*executiondomain.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*executiondomain.Execution, 0)
	for _, item := range r.items {
		if item.Status == executiondomain.StatusDispatched || item.Status == executiondomain.StatusRunning {
			result = append(result, clone(item))
		}
	}
	sortItems(result)
	return result, nil
}

func sortItems(items []*executiondomain.Execution) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
		return items[i].Priority > items[j].Priority
	})
}
func clone(item *executiondomain.Execution) *executiondomain.Execution {
	copy := *item
	copy.Labels = cloneMap(item.Labels)
	copy.Parameters = cloneMap(item.Parameters)
	copy.ReceiptIDs = append([]string(nil), item.ReceiptIDs...)
	copy.Events = append([]executiondomain.Event(nil), item.Events...)
	if item.LeaseExpiresAt != nil {
		value := *item.LeaseExpiresAt
		copy.LeaseExpiresAt = &value
	}
	if item.TimeoutAt != nil {
		value := *item.TimeoutAt
		copy.TimeoutAt = &value
	}
	return &copy
}
func cloneMap(source map[string]string) map[string]string {
	target := make(map[string]string, len(source))
	for k, v := range source {
		target[k] = v
	}
	return target
}
