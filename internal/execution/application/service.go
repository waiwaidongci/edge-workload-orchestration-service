package application

import (
	"context"
	"errors"
	"fmt"

	executiondomain "github.com/example/edge-task-orchestrator/internal/execution/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	"github.com/example/edge-task-orchestrator/internal/platform/id"
	policydomain "github.com/example/edge-task-orchestrator/internal/policy/domain"
	templatedomain "github.com/example/edge-task-orchestrator/internal/template/domain"
)

type SubmitCommand struct {
	TemplateID      string            `json:"template_id"`
	TemplateVersion int               `json:"template_version"`
	PolicyID        string            `json:"policy_id"`
	Priority        int               `json:"priority"`
	Labels          map[string]string `json:"labels"`
	Parameters      map[string]string `json:"parameters"`
}
type ReceiptCommand struct {
	ReceiptID string `json:"receipt_id"`
	State     string `json:"state"`
	Success   bool   `json:"success"`
	Reason    string `json:"reason"`
}

type Service struct {
	repository executiondomain.Repository
	templates  templatedomain.Repository
	policies   policydomain.Repository
	clock      clock.Clock
}

func NewService(repository executiondomain.Repository, templates templatedomain.Repository, policies policydomain.Repository, source clock.Clock) *Service {
	return &Service{repository: repository, templates: templates, policies: policies, clock: source}
}

func (s *Service) Submit(ctx context.Context, command SubmitCommand) (*executiondomain.Execution, error) {
	template, err := s.templates.Get(ctx, command.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	version, err := template.Version(command.TemplateVersion)
	if err != nil {
		return nil, fmt.Errorf("get template version: %w", err)
	}
	policy, err := s.policies.Get(ctx, command.PolicyID)
	if err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}
	if !policy.Enabled {
		return nil, fmt.Errorf("selected policy is disabled")
	}
	item, err := executiondomain.New(id.New("task"), template.ID, version.Number, policy.ID, command.Priority, command.Labels, command.Parameters, version.Resources, version.Retry.MaxAttempts, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("build execution: %w", err)
	}
	if err := s.repository.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("save execution: %w", err)
	}
	return item, nil
}

func (s *Service) Get(ctx context.Context, executionID string) (*executiondomain.Execution, error) {
	item, err := s.repository.Get(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	return item, nil
}
func (s *Service) List(ctx context.Context, filter executiondomain.Filter) ([]*executiondomain.Execution, error) {
	items, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list executions: %w", err)
	}
	return items, nil
}

func (s *Service) Cancel(ctx context.Context, executionID string) (*executiondomain.Execution, error) {
	item, err := s.repository.Get(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	if err := item.RequestCancel(s.clock.Now()); err != nil {
		return nil, err
	}
	if err := s.repository.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("update execution: %w", err)
	}
	return item, nil
}

func (s *Service) Receipt(ctx context.Context, executionID, nodeID string, command ReceiptCommand) (*executiondomain.Execution, error) {
	item, err := s.repository.Get(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	if item.NodeID != nodeID {
		return nil, fmt.Errorf("%w: receipt node does not own execution", executiondomain.ErrConflict)
	}
	now := s.clock.Now()
	switch command.State {
	case "running":
		err = item.Start(command.ReceiptID, now)
	case "completed":
		err = item.Complete(command.ReceiptID, command.Success, command.Reason, now)
	case "cancelled":
		err = item.ConfirmCancel(command.ReceiptID, now)
	default:
		err = fmt.Errorf("unknown receipt state %q", command.State)
	}
	if err != nil {
		return nil, err
	}
	if err := s.repository.Update(ctx, item); err != nil {
		if errors.Is(err, executiondomain.ErrConflict) {
			return s.repository.Get(ctx, executionID)
		}
		return nil, fmt.Errorf("update execution: %w", err)
	}
	return item, nil
}
