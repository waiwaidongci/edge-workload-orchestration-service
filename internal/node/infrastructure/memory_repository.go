package infrastructure

import (
	"context"
	"fmt"
	"sort"
	"sync"

	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*nodedomain.Node
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*nodedomain.Node)}
}

func (r *MemoryRepository) Save(ctx context.Context, node *nodedomain.Node) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[node.ID]; exists {
		return fmt.Errorf("node %s already exists", node.ID)
	}
	r.items[node.ID] = clone(node)
	return nil
}

func (r *MemoryRepository) Get(ctx context.Context, nodeID string) (*nodedomain.Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	node, ok := r.items[nodeID]
	if !ok {
		return nil, nodedomain.ErrNotFound
	}
	return clone(node), nil
}

func (r *MemoryRepository) List(ctx context.Context, filter nodedomain.Filter) ([]*nodedomain.Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*nodedomain.Node, 0, len(r.items))
	for _, node := range r.items {
		if filter.Region != "" && node.Region != filter.Region {
			continue
		}
		if filter.Status != "" && node.Status != filter.Status {
			continue
		}
		if !contains(node.Labels, filter.Labels) {
			continue
		}
		result = append(result, clone(node))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}

func (r *MemoryRepository) Update(ctx context.Context, node *nodedomain.Node) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[node.ID]; !exists {
		return nodedomain.ErrNotFound
	}
	r.items[node.ID] = clone(node)
	return nil
}

func clone(node *nodedomain.Node) *nodedomain.Node {
	copy := *node
	copy.Labels = map[string]string{}
	for k, v := range node.Labels {
		copy.Labels[k] = v
	}
	return &copy
}
func contains(have, want map[string]string) bool {
	for k, v := range want {
		if have[k] != v {
			return false
		}
	}
	return true
}
