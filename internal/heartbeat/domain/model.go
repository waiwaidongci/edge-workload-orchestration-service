package domain

import "time"

type Record struct {
	ID           string            `json:"id"`
	NodeID       string            `json:"node_id"`
	ObservedAt   time.Time         `json:"observed_at"`
	AgentUptime  int64             `json:"agent_uptime_seconds"`
	AgentVersion string            `json:"agent_version"`
	Metadata     map[string]string `json:"metadata"`
}

type StatusChange struct {
	NodeID string    `json:"node_id"`
	From   string    `json:"from"`
	To     string    `json:"to"`
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}
