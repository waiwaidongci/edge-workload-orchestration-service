package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	StatusOnline      Status = "online"
	StatusOffline     Status = "offline"
	StatusDraining    Status = "draining"
	StatusMaintenance Status = "maintenance"
)

var ErrNotFound = errors.New("node not found")

type Resources struct {
	CPUMillis   int64 `json:"cpu_millis"`
	MemoryMB    int64 `json:"memory_mb"`
	StorageMB   int64 `json:"storage_mb"`
	Accelerator int64 `json:"accelerator"`
}

func (r Resources) Validate() error {
	if r.CPUMillis < 0 || r.MemoryMB < 0 || r.StorageMB < 0 || r.Accelerator < 0 {
		return fmt.Errorf("resources cannot be negative")
	}
	return nil
}

func (r Resources) Add(other Resources) Resources {
	return Resources{r.CPUMillis + other.CPUMillis, r.MemoryMB + other.MemoryMB, r.StorageMB + other.StorageMB, r.Accelerator + other.Accelerator}
}

func (r Resources) Sub(other Resources) Resources {
	return Resources{r.CPUMillis - other.CPUMillis, r.MemoryMB - other.MemoryMB, r.StorageMB - other.StorageMB, r.Accelerator - other.Accelerator}
}

func (r Resources) Fits(request Resources) bool {
	return r.CPUMillis >= request.CPUMillis && r.MemoryMB >= request.MemoryMB && r.StorageMB >= request.StorageMB && r.Accelerator >= request.Accelerator
}

type Node struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Region        string            `json:"region"`
	Zone          string            `json:"zone"`
	Labels        map[string]string `json:"labels"`
	Capacity      Resources         `json:"capacity"`
	Allocated     Resources         `json:"allocated"`
	MaxConcurrent int               `json:"max_concurrent"`
	ActiveTasks   int               `json:"active_tasks"`
	Status        Status            `json:"status"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Version       int64             `json:"version"`
}

func New(id, name, region, zone string, labels map[string]string, capacity Resources, maxConcurrent int, now time.Time) (*Node, error) {
	n := &Node{ID: strings.TrimSpace(id), Name: strings.TrimSpace(name), Region: strings.TrimSpace(region), Zone: strings.TrimSpace(zone), Labels: cloneLabels(labels), Capacity: capacity, MaxConcurrent: maxConcurrent, Status: StatusOnline, LastHeartbeat: now, CreatedAt: now, UpdatedAt: now, Version: 1}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *Node) Validate() error {
	if n.ID == "" || n.Name == "" || n.Region == "" {
		return fmt.Errorf("node id, name and region are required")
	}
	if err := n.Capacity.Validate(); err != nil {
		return fmt.Errorf("capacity: %w", err)
	}
	if n.MaxConcurrent < 1 {
		return fmt.Errorf("max_concurrent must be positive")
	}
	return nil
}

func (n *Node) Available() Resources { return n.Capacity.Sub(n.Allocated) }

func (n *Node) CanRun(request Resources) bool {
	return n.Status == StatusOnline && n.ActiveTasks < n.MaxConcurrent && n.Available().Fits(request)
}

func (n *Node) Reserve(request Resources, now time.Time) error {
	if !n.CanRun(request) {
		return fmt.Errorf("node %s has insufficient capacity or is unavailable", n.ID)
	}
	n.Allocated = n.Allocated.Add(request)
	n.ActiveTasks++
	n.UpdatedAt = now
	n.Version++
	return nil
}

func (n *Node) Release(request Resources, now time.Time) {
	n.Allocated = n.Allocated.Sub(request)
	if n.Allocated.CPUMillis < 0 || n.Allocated.MemoryMB < 0 || n.Allocated.StorageMB < 0 || n.Allocated.Accelerator < 0 {
		n.Allocated = Resources{}
	}
	if n.ActiveTasks > 0 {
		n.ActiveTasks--
	}
	n.UpdatedAt = now
	n.Version++
}

func (n *Node) RecordHeartbeat(now time.Time) {
	n.LastHeartbeat = now
	if n.Status == StatusOffline {
		n.Status = StatusOnline
	}
	n.UpdatedAt = now
	n.Version++
}

func (n *Node) MarkOffline(now time.Time) { n.Status = StatusOffline; n.UpdatedAt = now; n.Version++ }

func cloneLabels(labels map[string]string) map[string]string {
	out := make(map[string]string, len(labels))
	for k, v := range labels {
		out[k] = v
	}
	return out
}
