package domain

import "time"

type DeadLetter struct {
	ID          string     `json:"id"`
	ExecutionID string     `json:"execution_id"`
	NodeID      string     `json:"node_id,omitempty"`
	Reason      string     `json:"reason"`
	Attempts    int        `json:"attempts"`
	Resolved    bool       `json:"resolved"`
	Resolution  string     `json:"resolution,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

func (d *DeadLetter) Resolve(note string, now time.Time) {
	d.Resolved = true
	d.Resolution = note
	d.ResolvedAt = &now
}

func Backoff(base time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		return base
	}
	if attempt > 8 {
		attempt = 8
	}
	delay := base
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	return delay
}
