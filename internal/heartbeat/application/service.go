package application

import (
	"context"
	"fmt"
	"time"

	heartbeatdomain "github.com/example/edge-task-orchestrator/internal/heartbeat/domain"
	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	"github.com/example/edge-task-orchestrator/internal/platform/id"
)

type BeatCommand struct {
	AgentUptime  int64             `json:"agent_uptime_seconds"`
	AgentVersion string            `json:"agent_version"`
	Metadata     map[string]string `json:"metadata"`
}
type Service struct {
	nodes      nodedomain.Repository
	repository heartbeatdomain.Repository
	presence   heartbeatdomain.Presence
	clock      clock.Clock
	timeout    time.Duration
}

func NewService(nodes nodedomain.Repository, repository heartbeatdomain.Repository, presence heartbeatdomain.Presence, source clock.Clock, timeout time.Duration) *Service {
	return &Service{nodes: nodes, repository: repository, presence: presence, clock: source, timeout: timeout}
}

func (s *Service) Beat(ctx context.Context, nodeID string, command BeatCommand) (heartbeatdomain.Record, error) {
	now := s.clock.Now()
	node, err := s.nodes.Get(ctx, nodeID)
	if err != nil {
		return heartbeatdomain.Record{}, fmt.Errorf("get heartbeat node: %w", err)
	}
	previous := node.Status
	node.RecordHeartbeat(now)
	if err := s.nodes.Update(ctx, node); err != nil {
		return heartbeatdomain.Record{}, fmt.Errorf("update node heartbeat: %w", err)
	}
	record := heartbeatdomain.Record{ID: id.New("heartbeat"), NodeID: nodeID, ObservedAt: now, AgentUptime: command.AgentUptime, AgentVersion: command.AgentVersion, Metadata: command.Metadata}
	if err := s.repository.Append(ctx, record); err != nil {
		return heartbeatdomain.Record{}, fmt.Errorf("store heartbeat: %w", err)
	}
	if err := s.presence.Touch(ctx, nodeID, s.timeout); err != nil {
		return heartbeatdomain.Record{}, fmt.Errorf("touch presence: %w", err)
	}
	if previous == nodedomain.StatusOffline {
		_ = s.repository.AppendStatusChange(ctx, heartbeatdomain.StatusChange{NodeID: nodeID, From: string(previous), To: string(node.Status), Reason: "heartbeat restored", At: now})
	}
	return record, nil
}

func (s *Service) Sweep(ctx context.Context) (int, error) {
	nodes, err := s.nodes.List(ctx, nodedomain.Filter{Status: nodedomain.StatusOnline})
	if err != nil {
		return 0, fmt.Errorf("list nodes for heartbeat sweep: %w", err)
	}
	now := s.clock.Now()
	changed := 0
	for _, node := range nodes {
		if now.Sub(node.LastHeartbeat) <= s.timeout {
			continue
		}
		previous := node.Status
		node.MarkOffline(now)
		if err := s.nodes.Update(ctx, node); err != nil {
			return changed, fmt.Errorf("mark node offline: %w", err)
		}
		if err := s.presence.Remove(ctx, node.ID); err != nil {
			return changed, fmt.Errorf("remove presence for node %s: %w", node.ID, err)
		}
		if err := s.repository.AppendStatusChange(ctx, heartbeatdomain.StatusChange{NodeID: node.ID, From: string(previous), To: string(node.Status), Reason: "heartbeat timeout", At: now}); err != nil {
			return changed, fmt.Errorf("store status change: %w", err)
		}
		changed++
	}
	return changed, nil
}

func (s *Service) Recent(ctx context.Context, nodeID string, limit int) ([]heartbeatdomain.Record, error) {
	return s.repository.Recent(ctx, nodeID, s.clock.Now().Add(-24*time.Hour), limit)
}
