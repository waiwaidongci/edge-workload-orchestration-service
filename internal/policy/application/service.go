package application

import (
	"context"
	"fmt"

	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	"github.com/example/edge-task-orchestrator/internal/platform/id"
	policydomain "github.com/example/edge-task-orchestrator/internal/policy/domain"
)

type CreateCommand struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Priority    int                      `json:"priority"`
	Constraints policydomain.Constraints `json:"constraints"`
}
type Service struct {
	repository policydomain.Repository
	clock      clock.Clock
}

func NewService(repository policydomain.Repository, source clock.Clock) *Service {
	return &Service{repository: repository, clock: source}
}

func (s *Service) Create(ctx context.Context, command CreateCommand) (*policydomain.Policy, error) {
	item, err := policydomain.New(id.New("policy"), command.Name, command.Description, command.Priority, command.Constraints, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("build policy: %w", err)
	}
	if err := s.repository.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("save policy: %w", err)
	}
	return item, nil
}
func (s *Service) Get(ctx context.Context, policyID string) (*policydomain.Policy, error) {
	item, err := s.repository.Get(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}
	return item, nil
}
func (s *Service) List(ctx context.Context) ([]*policydomain.Policy, error) {
	items, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	return items, nil
}
func (s *Service) SetEnabled(ctx context.Context, policyID string, enabled bool) (*policydomain.Policy, error) {
	item, err := s.repository.Get(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}
	item.SetEnabled(enabled, s.clock.Now())
	if err := s.repository.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("update policy: %w", err)
	}
	return item, nil
}
