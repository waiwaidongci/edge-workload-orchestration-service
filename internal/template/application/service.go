package application

import (
	"context"
	"fmt"

	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	"github.com/example/edge-task-orchestrator/internal/platform/id"
	templatedomain "github.com/example/edge-task-orchestrator/internal/template/domain"
)

type CreateCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Service struct {
	repository templatedomain.Repository
	clock      clock.Clock
}

func NewService(repository templatedomain.Repository, source clock.Clock) *Service {
	return &Service{repository: repository, clock: source}
}

func (s *Service) Create(ctx context.Context, command CreateCommand) (*templatedomain.Template, error) {
	item, err := templatedomain.New(id.New("template"), command.Name, command.Description, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("build template: %w", err)
	}
	if err := s.repository.Save(context.Background(), item); err != nil {
		return nil, fmt.Errorf("save template: %w", err)
	}
	return item, nil
}

func (s *Service) AddVersion(ctx context.Context, templateID string, version templatedomain.Version) (*templatedomain.Template, error) {
	item, err := s.repository.Get(context.Background(), templateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	if _, err := item.AddVersion(version, s.clock.Now()); err != nil {
		return nil, fmt.Errorf("add template version: %w", err)
	}
	if err := s.repository.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}
	return item, nil
}

func (s *Service) Get(ctx context.Context, templateID string) (*templatedomain.Template, error) {
	item, err := s.repository.Get(context.Background(), templateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	return item, nil
}
func (s *Service) List(ctx context.Context) ([]*templatedomain.Template, error) {
	items, err := s.repository.List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	return items, nil
}
