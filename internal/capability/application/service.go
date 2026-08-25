package application

import (
	"context"
	"fmt"

	capabilitydomain "github.com/example/edge-task-orchestrator/internal/capability/domain"
	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	"github.com/example/edge-task-orchestrator/internal/platform/id"
)

type UpsertCommand struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Attributes map[string]string `json:"attributes"`
	Discovered bool              `json:"discovered"`
}

type Service struct {
	repository capabilitydomain.Repository
	nodes      nodedomain.Repository
	clock      clock.Clock
}

func NewService(repository capabilitydomain.Repository, nodes nodedomain.Repository, source clock.Clock) *Service {
	return &Service{repository: repository, nodes: nodes, clock: source}
}

func (s *Service) Upsert(ctx context.Context, nodeID string, command UpsertCommand) (*capabilitydomain.Capability, error) {
	if _, err := s.nodes.Get(ctx, nodeID); err != nil {
		return nil, fmt.Errorf("get capability node: %w", err)
	}
	item, err := capabilitydomain.New(id.New("cap"), nodeID, command.Name, command.Version, command.Attributes, command.Discovered, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("build capability: %w", err)
	}
	if err := s.repository.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("save capability: %w", err)
	}
	return item, nil
}

func (s *Service) List(ctx context.Context, nodeID string) ([]*capabilitydomain.Capability, error) {
	items, err := s.repository.ListByNode(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list capabilities: %w", err)
	}
	return items, nil
}

func (s *Service) Delete(ctx context.Context, nodeID, name string) error {
	if err := s.repository.Delete(ctx, nodeID, name); err != nil {
		return fmt.Errorf("delete capability: %w", err)
	}
	return nil
}
