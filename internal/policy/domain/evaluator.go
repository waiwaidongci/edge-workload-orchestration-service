package domain

import (
	"fmt"

	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
)

type Evaluation struct {
	Eligible bool     `json:"eligible"`
	Score    int      `json:"score"`
	Reasons  []string `json:"reasons"`
}

func Evaluate(policy *Policy, node *nodedomain.Node, resources nodedomain.Resources, taskLabels map[string]string) Evaluation {
	if policy == nil || node == nil {
		return Evaluation{Eligible: false, Reasons: []string{"policy or node is nil"}}
	}
	result := Evaluation{Eligible: true, Score: policy.Priority, Reasons: []string{}}
	if !policy.Enabled {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "policy is disabled")
		return result
	}
	if !node.CanRun(resources) {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "node unavailable or lacks resources")
	}
	if policy.Constraints.MaxTasksPerNode > 0 && node.ActiveTasks >= policy.Constraints.MaxTasksPerNode {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "policy concurrency limit reached")
	}
	for key, value := range policy.Constraints.RequiredLabels {
		if node.Labels[key] != value {
			result.Eligible = false
			result.Reasons = append(result.Reasons, fmt.Sprintf("required label %s=%s missing", key, value))
		}
	}
	if len(policy.Constraints.RequiredRegions) > 0 && !includes(policy.Constraints.RequiredRegions, node.Region) {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "node region not allowed")
	}
	if includes(policy.Constraints.PreferredRegions, node.Region) {
		result.Score += 100
		result.Reasons = append(result.Reasons, "preferred region")
	}
	for key, value := range policy.Constraints.AffinityLabels {
		if node.Labels[key] == value || taskLabels[key] == value {
			result.Score += 20
			result.Reasons = append(result.Reasons, "affinity matched "+key)
		}
	}
	for key, value := range policy.Constraints.AntiAffinityLabels {
		if node.Labels[key] == value && taskLabels[key] == value {
			result.Score -= 30
			result.Reasons = append(result.Reasons, "anti-affinity penalty "+key)
		}
	}
	available := node.Available()
	result.Score += int(available.CPUMillis/100 + available.MemoryMB/256)
	return result
}

func includes(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
