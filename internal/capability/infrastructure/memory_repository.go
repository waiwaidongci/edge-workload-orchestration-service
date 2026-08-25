package infrastructure

import (
	"context"
	"sort"
	"sync"

	capabilitydomain "github.com/example/edge-task-orchestrator/internal/capability/domain"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]map[string]*capabilitydomain.Capability
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]map[string]*capabilitydomain.Capability)}
}

func (r *MemoryRepository) Save(ctx context.Context, item *capabilitydomain.Capability) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.items[item.NodeID] == nil {
		r.items[item.NodeID] = make(map[string]*capabilitydomain.Capability)
	}
	stored := clone(item)
	r.items[item.NodeID][item.Name] = stored
	return nil
}

func (r *MemoryRepository) ListByNode(ctx context.Context, nodeID string) ([]*capabilitydomain.Capability, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*capabilitydomain.Capability, 0, len(r.items[nodeID]))
	for _, item := range r.items[nodeID] {
		result = append(result, clone(item))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (r *MemoryRepository) Find(ctx context.Context, nodeID, name string) (*capabilitydomain.Capability, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, exists := r.items[nodeID][name]
	if !exists {
		return nil, capabilitydomain.ErrNotFound
	}
	result := clone(item)
	return result, nil
}

func (r *MemoryRepository) Delete(ctx context.Context, nodeID, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[nodeID][name]; !exists {
		return capabilitydomain.ErrNotFound
	}
	delete(r.items[nodeID], name)
	return nil
}

func clone(item *capabilitydomain.Capability) *capabilitydomain.Capability {
	copy := *item
	copy.Attributes = make(map[string]string, len(item.Attributes))
	for k, v := range item.Attributes {
		copy.Attributes[k] = v
	}
	return &copy
}
