package infrastructure

import (
	"context"
	"fmt"
	"sort"
	"sync"

	templatedomain "github.com/example/edge-task-orchestrator/internal/template/domain"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*templatedomain.Template
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*templatedomain.Template)}
}

func (r *MemoryRepository) Save(ctx context.Context, item *templatedomain.Template) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[item.ID]; exists {
		return fmt.Errorf("template %s already exists", item.ID)
	}
	r.items[item.ID] = clone(item)
	return nil
}

func (r *MemoryRepository) Get(ctx context.Context, templateID string) (*templatedomain.Template, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, exists := r.items[templateID]
	if !exists {
		return nil, templatedomain.ErrNotFound
	}
	return clone(item), nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]*templatedomain.Template, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*templatedomain.Template, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, clone(item))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}

func (r *MemoryRepository) Update(ctx context.Context, item *templatedomain.Template) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[item.ID]; !exists {
		return templatedomain.ErrNotFound
	}
	r.items[item.ID] = clone(item)
	return nil
}

func clone(item *templatedomain.Template) *templatedomain.Template {
	copy := *item
	copy.Versions = item.Versions
	return &copy
}
