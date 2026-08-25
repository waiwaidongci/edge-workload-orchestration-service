package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("capability not found")

type Capability struct {
	ID         string            `json:"id"`
	NodeID     string            `json:"node_id"`
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Attributes map[string]string `json:"attributes"`
	Enabled    bool              `json:"enabled"`
	Discovered bool              `json:"discovered"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

func New(identifier, nodeID, name, version string, attributes map[string]string, discovered bool, now time.Time) (*Capability, error) {
	capability := &Capability{ID: identifier, NodeID: strings.TrimSpace(nodeID), Name: strings.TrimSpace(name), Version: strings.TrimSpace(version), Attributes: clone(attributes), Enabled: true, Discovered: discovered, CreatedAt: now, UpdatedAt: now}
	if capability.NodeID == "" || capability.Name == "" {
		return nil, fmt.Errorf("node_id and name are required")
	}
	if capability.Version == "" {
		capability.Version = "unknown"
	}
	return capability, nil
}

func (c *Capability) Disable(now time.Time) { c.Enabled = false; c.UpdatedAt = now }
func (c *Capability) Enable(now time.Time)  { c.Enabled = true; c.UpdatedAt = now }
func clone(source map[string]string) map[string]string {
	target := make(map[string]string, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}
