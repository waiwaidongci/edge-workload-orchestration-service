package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("scheduling policy not found")

type Constraints struct {
	RequiredLabels     map[string]string `json:"required_labels"`
	PreferredRegions   []string          `json:"preferred_regions"`
	RequiredRegions    []string          `json:"required_regions"`
	AffinityLabels     map[string]string `json:"affinity_labels"`
	AntiAffinityLabels map[string]string `json:"anti_affinity_labels"`
	MaxTasksPerNode    int               `json:"max_tasks_per_node"`
}

type Policy struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Priority    int         `json:"priority"`
	Enabled     bool        `json:"enabled"`
	Constraints Constraints `json:"constraints"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func New(identifier, name, description string, priority int, constraints Constraints, now time.Time) (*Policy, error) {
	item := &Policy{ID: strings.TrimSpace(identifier), Name: strings.TrimSpace(name), Description: strings.TrimSpace(description), Priority: priority, Enabled: true, Constraints: copyConstraints(constraints), CreatedAt: now, UpdatedAt: now}
	if item.ID == "" || item.Name == "" {
		return nil, fmt.Errorf("policy id and name are required")
	}
	if priority < -1000 || priority > 1000 {
		return nil, fmt.Errorf("priority must be between -1000 and 1000")
	}
	if constraints.MaxTasksPerNode < 0 {
		return nil, fmt.Errorf("max_tasks_per_node cannot be negative")
	}
	return item, nil
}

func (p *Policy) SetEnabled(enabled bool, now time.Time) { p.Enabled = enabled; p.UpdatedAt = now }

func copyConstraints(source Constraints) Constraints {
	target := source
	target.RequiredLabels = copyMap(source.RequiredLabels)
	target.AffinityLabels = copyMap(source.AffinityLabels)
	target.AntiAffinityLabels = copyMap(source.AntiAffinityLabels)
	target.PreferredRegions = source.PreferredRegions
	target.RequiredRegions = source.RequiredRegions
	return target
}
func copyMap(source map[string]string) map[string]string {
	target := make(map[string]string, len(source))
	for k, v := range source {
		target[k] = v
	}
	return target
}
