package infrastructure

import (
	"context"
	"errors"
	"sort"
	"sync"

	failuredomain "github.com/example/edge-task-orchestrator/internal/failure/domain"
)

var ErrNotFound = errors.New("dead letter not found")

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*failuredomain.DeadLetter
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*failuredomain.DeadLetter)}
}
func (r *MemoryRepository) Save(ctx context.Context, item *failuredomain.DeadLetter) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *item
	r.items[item.ID] = &copy
	return nil
}
func (r *MemoryRepository) Get(ctx context.Context, id string) (*failuredomain.DeadLetter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	copy := *item
	return &copy, nil
}
func (r *MemoryRepository) List(ctx context.Context, includeResolved bool, limit int) ([]*failuredomain.DeadLetter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*failuredomain.DeadLetter, 0)
	for _, item := range r.items {
		if item.Resolved && !includeResolved {
			continue
		}
		copy := *item
		result = append(result, &copy)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
func (r *MemoryRepository) Update(ctx context.Context, item *failuredomain.DeadLetter) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; !ok {
		return ErrNotFound
	}
	copy := *item
	r.items[item.ID] = &copy
	return nil
}
