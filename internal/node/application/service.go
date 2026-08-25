package application

import (
	"context"
	"fmt"
	"strings"

	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	"github.com/example/edge-task-orchestrator/internal/platform/id"
)

type RegisterCommand struct {
	Name          string               `json:"name"`
	Region        string               `json:"region"`
	Zone          string               `json:"zone"`
	Labels        map[string]string    `json:"labels"`
	Capacity      nodedomain.Resources `json:"capacity"`
	MaxConcurrent int                  `json:"max_concurrent"`
}

type Service struct {
	repository nodedomain.Repository
	clock      clock.Clock
}

func NewService(repository nodedomain.Repository, source clock.Clock) *Service {
	return &Service{repository: repository, clock: source}
}

func (s *Service) Register(ctx context.Context, command RegisterCommand) (*nodedomain.Node, error) {
	node, err := nodedomain.New(id.New("node"), command.Name, command.Region, command.Zone, command.Labels, command.Capacity, command.MaxConcurrent, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("build node: %w", err)
	}
	if err := s.repository.Save(ctx, node); err != nil {
		return nil, fmt.Errorf("save node: %w", err)
	}
	return node, nil
}

func (s *Service) Get(ctx context.Context, nodeID string) (*nodedomain.Node, error) {
	if strings.TrimSpace(nodeID) == "" {
		return nil, fmt.Errorf("node id is required")
	}
	node, err := s.repository.Get(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	return node, nil
}

func (s *Service) List(ctx context.Context, filter nodedomain.Filter) ([]*nodedomain.Node, error) {
	items, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	return items, nil
}
