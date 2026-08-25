package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
)

type Status string

const (
	StatusQueued     Status = "queued"
	StatusDispatched Status = "dispatched"
	StatusRunning    Status = "running"
	StatusSucceeded  Status = "succeeded"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

var ErrNotFound = errors.New("execution not found")
var ErrConflict = errors.New("execution state conflict")

type Event struct {
	Type    string    `json:"type"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

type Execution struct {
	ID              string               `json:"id"`
	TemplateID      string               `json:"template_id"`
	TemplateVersion int                  `json:"template_version"`
	PolicyID        string               `json:"policy_id"`
	NodeID          string               `json:"node_id,omitempty"`
	Priority        int                  `json:"priority"`
	Labels          map[string]string    `json:"labels"`
	Parameters      map[string]string    `json:"parameters"`
	Resources       nodedomain.Resources `json:"resources"`
	Status          Status               `json:"status"`
	Attempt         int                  `json:"attempt"`
	MaxAttempts     int                  `json:"max_attempts"`
	ScheduledAt     time.Time            `json:"scheduled_at"`
	LeaseExpiresAt  *time.Time           `json:"lease_expires_at,omitempty"`
	TimeoutAt       *time.Time           `json:"timeout_at,omitempty"`
	CancelRequested bool                 `json:"cancel_requested"`
	FailureReason   string               `json:"failure_reason,omitempty"`
	ReceiptIDs      []string             `json:"receipt_ids,omitempty"`
	Events          []Event              `json:"events"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Version         int64                `json:"version"`
}

func New(identifier, templateID string, templateVersion int, policyID string, priority int, labels, parameters map[string]string, resources nodedomain.Resources, maxAttempts int, now time.Time) (*Execution, error) {
	item := &Execution{ID: strings.TrimSpace(identifier), TemplateID: strings.TrimSpace(templateID), TemplateVersion: templateVersion, PolicyID: strings.TrimSpace(policyID), Priority: priority, Labels: cloneMap(labels), Parameters: cloneMap(parameters), Resources: resources, Status: StatusQueued, MaxAttempts: maxAttempts, ScheduledAt: now, CreatedAt: now, UpdatedAt: now, Version: 1}
	if item.ID == "" || item.TemplateID == "" || item.PolicyID == "" {
		return nil, fmt.Errorf("execution id, template_id and policy_id are required")
	}
	if item.TemplateVersion <= 0 {
		return nil, fmt.Errorf("template_version must be positive")
	}
	if item.MaxAttempts <= 0 {
		item.MaxAttempts = 1
	}
	if err := resources.Validate(); err != nil {
		return nil, fmt.Errorf("resources: %w", err)
	}
	item.record("queued", "task accepted", now)
	return item, nil
}

func (e *Execution) Dispatch(nodeID string, lease, timeout time.Duration, now time.Time) error {
	if e.Status != StatusQueued {
		return fmt.Errorf("%w: cannot dispatch from %s", ErrConflict, e.Status)
	}
	if strings.TrimSpace(nodeID) == "" {
		return fmt.Errorf("node id is required")
	}
	e.NodeID = nodeID
	e.Status = StatusDispatched
	e.Attempt++
	leaseEnd, timeoutEnd := now.Add(lease), now.Add(timeout)
	e.LeaseExpiresAt, e.TimeoutAt = &leaseEnd, &timeoutEnd
	e.FailureReason = ""
	e.touch(now)
	e.record("dispatched", "task sent to node "+nodeID, now)
	return nil
}

func (e *Execution) Start(receiptID string, now time.Time) error {
	if e.hasReceipt(receiptID) {
		return nil
	}
	if e.Status != StatusDispatched {
		return fmt.Errorf("%w: cannot start from %s", ErrConflict, e.Status)
	}
	e.addReceipt(receiptID)
	e.Status = StatusRunning
	e.touch(now)
	e.record("running", "node acknowledged task start", now)
	return nil
}

func (e *Execution) Complete(receiptID string, success bool, reason string, now time.Time) error {
	if e.hasReceipt(receiptID) {
		return nil
	}
	if e.Status != StatusRunning && e.Status != StatusDispatched {
		return fmt.Errorf("%w: cannot complete from %s", ErrConflict, e.Status)
	}
	e.addReceipt(receiptID)
	if success {
		e.Status = StatusSucceeded
		e.FailureReason = ""
		e.record("succeeded", "task completed", now)
	} else {
		e.Status = StatusFailed
		e.FailureReason = strings.TrimSpace(reason)
		e.record("failed", e.FailureReason, now)
	}
	e.LeaseExpiresAt = nil
	e.TimeoutAt = nil
	e.touch(now)
	return nil
}

func (e *Execution) RequestCancel(now time.Time) error {
	if e.Terminal() {
		if e.Status == StatusCancelled {
			return nil
		}
		return fmt.Errorf("%w: terminal execution cannot be cancelled", ErrConflict)
	}
	e.CancelRequested = true
	if e.Status == StatusQueued {
		e.Status = StatusCancelled
		e.record("cancelled", "cancelled before dispatch", now)
	} else {
		e.record("cancel_requested", "cancellation requested", now)
	}
	e.touch(now)
	return nil
}

func (e *Execution) ConfirmCancel(receiptID string, now time.Time) error {
	if e.hasReceipt(receiptID) {
		return nil
	}
	if !e.CancelRequested {
		return fmt.Errorf("%w: cancellation was not requested", ErrConflict)
	}
	e.addReceipt(receiptID)
	e.Status = StatusCancelled
	e.LeaseExpiresAt = nil
	e.TimeoutAt = nil
	e.touch(now)
	e.record("cancelled", "node confirmed cancellation", now)
	return nil
}

func (e *Execution) Requeue(reason string, delay time.Duration, now time.Time) error {
	if e.Attempt >= e.MaxAttempts {
		return fmt.Errorf("%w: attempts exhausted", ErrConflict)
	}
	e.Status = StatusQueued
	e.NodeID = ""
	e.LeaseExpiresAt = nil
	e.TimeoutAt = nil
	e.ScheduledAt = now.Add(delay)
	e.FailureReason = reason
	e.touch(now)
	e.record("requeued", reason, now)
	return nil
}

func (e *Execution) MarkFailed(reason string, now time.Time) {
	e.Status = StatusFailed
	e.FailureReason = reason
	e.LeaseExpiresAt = nil
	e.TimeoutAt = nil
	e.touch(now)
	e.record("failed", reason, now)
}
func (e *Execution) Runnable(now time.Time) bool {
	return e.Status == StatusQueued && !e.CancelRequested && !e.ScheduledAt.After(now)
}
func (e *Execution) Terminal() bool {
	return e.Status == StatusSucceeded || e.Status == StatusFailed || e.Status == StatusCancelled
}
func (e *Execution) LeaseExpired(now time.Time) bool {
	return (e.Status == StatusDispatched || e.Status == StatusRunning) && e.LeaseExpiresAt != nil && !now.Before(*e.LeaseExpiresAt)
}
func (e *Execution) TimedOut(now time.Time) bool {
	return !e.Terminal() && e.TimeoutAt != nil && !now.Before(*e.TimeoutAt)
}

func (e *Execution) touch(now time.Time) { e.UpdatedAt = now; e.Version++ }
func (e *Execution) record(kind, message string, now time.Time) {
	e.Events = append(e.Events, Event{Type: kind, Message: message, At: now})
}
func (e *Execution) hasReceipt(id string) bool {
	if id == "" {
		return false
	}
	for _, seen := range e.ReceiptIDs {
		if seen == id {
			return true
		}
	}
	return false
}
func (e *Execution) addReceipt(id string) {
	if strings.TrimSpace(id) != "" {
		e.ReceiptIDs = append(e.ReceiptIDs, id)
	}
}
func cloneMap(source map[string]string) map[string]string {
	target := make(map[string]string, len(source))
	for k, v := range source {
		target[k] = v
	}
	return target
}
