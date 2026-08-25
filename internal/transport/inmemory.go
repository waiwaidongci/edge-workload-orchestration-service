package transport

import (
	"context"
	"sync"

	executiondomain "github.com/example/edge-task-orchestrator/internal/execution/domain"
)

type Message struct {
	ExecutionID string `json:"execution_id"`
	NodeID      string `json:"node_id"`
	Attempt     int    `json:"attempt"`
	SentAt      int64  `json:"sent_at"`
}
type InMemoryDispatcher struct {
	mu       sync.RWMutex
	messages []Message
}

func NewInMemoryDispatcher() *InMemoryDispatcher { return &InMemoryDispatcher{} }
func (d *InMemoryDispatcher) Dispatch(ctx context.Context, item *executiondomain.Execution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, Message{ExecutionID: item.ID, NodeID: item.NodeID, Attempt: item.Attempt, SentAt: item.UpdatedAt.UnixMilli()})
	return nil
}
func (d *InMemoryDispatcher) Messages() []Message {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return append([]Message(nil), d.messages...)
}
