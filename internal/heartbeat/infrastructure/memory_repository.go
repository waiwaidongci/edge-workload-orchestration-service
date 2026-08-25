package infrastructure

import (
	"context"
	"sort"
	"sync"
	"time"

	heartbeatdomain "github.com/example/edge-task-orchestrator/internal/heartbeat/domain"
)

type MemoryRepository struct {
	mu      sync.RWMutex
	records map[string][]heartbeatdomain.Record
	changes []heartbeatdomain.StatusChange
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{records: make(map[string][]heartbeatdomain.Record)}
}

func (r *MemoryRepository) Append(ctx context.Context, record heartbeatdomain.Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record.Metadata = clone(record.Metadata)
	r.records[record.NodeID] = append(r.records[record.NodeID], record)
	if len(r.records[record.NodeID]) > 256 {
		r.records[record.NodeID] = append([]heartbeatdomain.Record(nil), r.records[record.NodeID][len(r.records[record.NodeID])-256:]...)
	}
	return nil
}
func (r *MemoryRepository) Recent(ctx context.Context, nodeID string, since time.Time, limit int) ([]heartbeatdomain.Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]heartbeatdomain.Record, 0)
	for _, item := range r.records[nodeID] {
		if !item.ObservedAt.Before(since) {
			item.Metadata = clone(item.Metadata)
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ObservedAt.After(result[j].ObservedAt) })
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
func (r *MemoryRepository) AppendStatusChange(ctx context.Context, change heartbeatdomain.StatusChange) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.changes = append(r.changes, change)
	return nil
}
func (r *MemoryRepository) StatusChanges(ctx context.Context, nodeID string, limit int) ([]heartbeatdomain.StatusChange, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]heartbeatdomain.StatusChange, 0)
	for i := len(r.changes) - 1; i >= 0; i-- {
		if nodeID == "" || r.changes[i].NodeID == nodeID {
			result = append(result, r.changes[i])
			if limit > 0 && len(result) == limit {
				break
			}
		}
	}
	return result, nil
}
func clone(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

type MemoryPresence struct {
	mu      sync.RWMutex
	expires map[string]time.Time
	now     func() time.Time
}

func NewMemoryPresence() *MemoryPresence {
	return &MemoryPresence{expires: make(map[string]time.Time), now: func() time.Time { return time.Now().UTC() }}
}
func (p *MemoryPresence) Touch(ctx context.Context, nodeID string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.expires[nodeID] = p.now().Add(ttl)
	return nil
}
func (p *MemoryPresence) Online(ctx context.Context, nodeID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	expires, ok := p.expires[nodeID]
	return ok && p.now().Before(expires), nil
}
func (p *MemoryPresence) Remove(ctx context.Context, nodeID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(p.expires, nodeID)
	return nil
}
