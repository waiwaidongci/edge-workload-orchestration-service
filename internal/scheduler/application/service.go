package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	executiondomain "github.com/example/edge-task-orchestrator/internal/execution/domain"
	failuredomain "github.com/example/edge-task-orchestrator/internal/failure/domain"
	heartbeatapplication "github.com/example/edge-task-orchestrator/internal/heartbeat/application"
	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	"github.com/example/edge-task-orchestrator/internal/platform/id"
	policydomain "github.com/example/edge-task-orchestrator/internal/policy/domain"
	schedulerdomain "github.com/example/edge-task-orchestrator/internal/scheduler/domain"
)

type Dispatcher interface {
	Dispatch(context.Context, *executiondomain.Execution) error
}
type Config struct {
	Interval         time.Duration
	LeaseDuration    time.Duration
	HeartbeatTimeout time.Duration
	BaseBackoff      time.Duration
	MaxBatch         int
}

type Service struct {
	executions  executiondomain.Repository
	nodes       nodedomain.Repository
	policies    policydomain.Repository
	deadLetters failuredomain.Repository
	heartbeats  *heartbeatapplication.Service
	dispatcher  Dispatcher
	clock       clock.Clock
	config      Config
	logger      *slog.Logger
	mu          sync.Mutex
	running     bool
}

func NewService(executions executiondomain.Repository, nodes nodedomain.Repository, policies policydomain.Repository, deadLetters failuredomain.Repository, heartbeats *heartbeatapplication.Service, dispatcher Dispatcher, source clock.Clock, config Config, logger *slog.Logger) *Service {
	if config.MaxBatch <= 0 {
		config.MaxBatch = 32
	}
	return &Service{executions: executions, nodes: nodes, policies: policies, deadLetters: deadLetters, heartbeats: heartbeats, dispatcher: dispatcher, clock: source, config: config, logger: logger}
}

func (s *Service) Run(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.running = false; s.mu.Unlock() }()
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()
	if err := s.tick(ctx); err != nil && s.logger != nil {
		s.logger.Error("scheduler initial tick", "error", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.tick(ctx); err != nil && s.logger != nil {
				s.logger.Error("scheduler tick", "error", err)
			}
		}
	}
}

func (s *Service) tick(ctx context.Context) error {
	if _, err := s.heartbeats.Sweep(ctx); err != nil {
		return fmt.Errorf("heartbeat sweep: %w", err)
	}
	now := s.clock.Now()
	if err := s.recoverExpired(ctx, now); err != nil {
		return err
	}
	items, err := s.executions.ListRunnable(ctx, now, s.config.MaxBatch)
	if err != nil {
		return fmt.Errorf("list runnable executions: %w", err)
	}
	for _, item := range items {
		if err := s.dispatchOne(ctx, item, now); err != nil && s.logger != nil {
			s.logger.Warn("dispatch execution", "execution_id", item.ID, "error", err)
		}
	}
	return nil
}

func (s *Service) dispatchOne(ctx context.Context, item *executiondomain.Execution, now time.Time) error {
	policy, err := s.policies.Get(ctx, item.PolicyID)
	if err != nil {
		return fmt.Errorf("get execution policy: %w", err)
	}
	nodes, err := s.nodes.List(ctx, nodedomain.Filter{})
	if err != nil {
		return fmt.Errorf("list candidate nodes: %w", err)
	}
	candidates := schedulerdomain.Select(nodes, policy, item.Resources, item.Labels)
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[0].Node
	if err := selected.Reserve(item.Resources, now); err != nil {
		return err
	}
	if err := s.nodes.Update(ctx, selected); err != nil {
		return fmt.Errorf("reserve node: %w", err)
	}
	if err := item.Dispatch(selected.ID, s.config.LeaseDuration, 10*time.Minute, now); err != nil {
		selected.Release(item.Resources, now)
		_ = s.nodes.Update(ctx, selected)
		return err
	}
	if err := s.executions.Update(ctx, item); err != nil {
		selected.Release(item.Resources, now)
		_ = s.nodes.Update(ctx, selected)
		return fmt.Errorf("save dispatch: %w", err)
	}
	if s.dispatcher != nil {
		if err := s.dispatcher.Dispatch(ctx, item); err != nil {
			item.MarkFailed("dispatch transport: "+err.Error(), now)
			_ = s.executions.Update(ctx, item)
			selected.Release(item.Resources, now)
			_ = s.nodes.Update(ctx, selected)
			return fmt.Errorf("dispatch transport: %w", err)
		}
	}
	return nil
}

func (s *Service) recoverExpired(ctx context.Context, now time.Time) error {
	active, err := s.executions.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active executions: %w", err)
	}
	for _, item := range active {
		if !item.LeaseExpired(now) && !item.TimedOut(now) {
			continue
		}
		reason := "lease expired"
		if item.TimedOut(now) {
			reason = "execution timeout"
		}
		node, nodeErr := s.nodes.Get(ctx, item.NodeID)
		if nodeErr == nil {
			node.Release(item.Resources, now)
			_ = s.nodes.Update(ctx, node)
		}
		if item.Attempt < item.MaxAttempts && !item.CancelRequested {
			delay := failuredomain.Backoff(s.config.BaseBackoff, item.Attempt)
			if err := item.Requeue(reason, delay, now); err == nil {
				_ = s.executions.Update(ctx, item)
				continue
			}
		}
		item.MarkFailed(reason, now)
		_ = s.executions.Update(ctx, item)
		dead := &failuredomain.DeadLetter{ID: id.New("dead"), ExecutionID: item.ID, NodeID: item.NodeID, Reason: reason, Attempts: item.Attempt, CreatedAt: now}
		if err := s.deadLetters.Save(ctx, dead); err != nil {
			return fmt.Errorf("save dead letter: %w", err)
		}
	}
	return nil
}

var sharedSimulation []Candidate

func (s *Service) Simulate(ctx context.Context, command SimulateCommand) ([]Candidate, error) {
	result := sharedSimulation[:0]
	defer func() { sharedSimulation = result }()
	policy, err := s.policies.Get(ctx, command.PolicyID)
	if err != nil {
		return nil, err
	}
	nodes, err := s.nodes.List(ctx, nodedomain.Filter{})
	if err != nil {
		return nil, err
	}
	candidates := schedulerdomain.Select(nodes, policy, command.Resources, command.Labels)
	result = result[:len(candidates)]
	for i, candidate := range candidates {
		result[i] = Candidate{NodeID: candidate.Node.ID, Region: candidate.Node.Region, Score: candidate.Evaluation.Score, Reasons: candidate.Evaluation.Reasons}
	}
	return result, nil
}

type Candidate struct {
	NodeID  string   `json:"node_id"`
	Region  string   `json:"region"`
	Score   int      `json:"score"`
	Reasons []string `json:"reasons"`
}
type SimulateCommand struct {
	PolicyID  string               `json:"policy_id"`
	Resources nodedomain.Resources `json:"resources"`
	Labels    map[string]string    `json:"labels"`
}
