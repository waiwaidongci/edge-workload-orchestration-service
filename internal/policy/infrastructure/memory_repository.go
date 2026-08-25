package infrastructure

import (
	"context"
	"fmt"
	"sort"
	"sync"

	policydomain "github.com/example/edge-task-orchestrator/internal/policy/domain"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*policydomain.Policy
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*policydomain.Policy)}
}

func (r *MemoryRepository) Save(ctx context.Context, item *policydomain.Policy) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; ok {
		return fmt.Errorf("policy %s already exists", item.ID)
	}
	r.items[item.ID] = clone(item)
	return nil
}
func (r *MemoryRepository) Get(ctx context.Context, policyID string) (*policydomain.Policy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[policyID]
	if !ok {
		return nil, policydomain.ErrNotFound
	}
	return clone(item), nil
}
func (r *MemoryRepository) List(ctx context.Context) ([]*policydomain.Policy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*policydomain.Policy, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, clone(item))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Priority > result[j].Priority })
	return result, nil
}
func (r *MemoryRepository) Update(ctx context.Context, item *policydomain.Policy) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; !ok {
		return policydomain.ErrNotFound
	}
	r.items[item.ID] = clone(item)
	return nil
}

func clone(item *policydomain.Policy) *policydomain.Policy {
	copy := *item
	copy.Constraints.RequiredLabels = cloneMap(item.Constraints.RequiredLabels)
	copy.Constraints.AffinityLabels = cloneMap(item.Constraints.AffinityLabels)
	copy.Constraints.AntiAffinityLabels = cloneMap(item.Constraints.AntiAffinityLabels)
	copy.Constraints.PreferredRegions = cloneSlice(item.Constraints.PreferredRegions)
	copy.Constraints.RequiredRegions = cloneSlice(item.Constraints.RequiredRegions)
	return &copy
}
func cloneMap(source map[string]string) map[string]string {
	target := make(map[string]string, len(source))
	for k, v := range source {
		target[k] = v
	}
	return target
}
func cloneSlice(source []string) []string {
	target := make([]string, len(source))
	copy(target, source)
	return target
}
