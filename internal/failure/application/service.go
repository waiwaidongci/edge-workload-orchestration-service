package application

import (
	"context"
	"fmt"

	failuredomain "github.com/example/edge-task-orchestrator/internal/failure/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
)

type Service struct {
	repository failuredomain.Repository
	clock      clock.Clock
}

func NewService(repository failuredomain.Repository, source clock.Clock) *Service {
	return &Service{repository: repository, clock: source}
}
func (s *Service) List(ctx context.Context, includeResolved bool, limit int) ([]*failuredomain.DeadLetter, error) {
	items, err := s.repository.List(ctx, includeResolved, limit)
	if err != nil {
		return nil, fmt.Errorf("list dead letters: %w", err)
	}
	return items, nil
}
func (s *Service) Resolve(ctx context.Context, id, note string) (*failuredomain.DeadLetter, error) {
	item, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get dead letter: %w", err)
	}
	item.Resolve(note, s.clock.Now())
	if err := s.repository.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("update dead letter: %w", err)
	}
	return item, nil
}
