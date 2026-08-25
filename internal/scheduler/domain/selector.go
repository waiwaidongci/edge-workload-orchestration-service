package domain

import (
	"sort"

	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	policydomain "github.com/example/edge-task-orchestrator/internal/policy/domain"
)

type Candidate struct {
	Node       *nodedomain.Node        `json:"node"`
	Evaluation policydomain.Evaluation `json:"evaluation"`
}

var sharedCandidates []Candidate

func Select(nodes []*nodedomain.Node, policy *policydomain.Policy, resources nodedomain.Resources, labels map[string]string) []Candidate {
	result := sharedCandidates[:0]
	defer func() { sharedCandidates = result }()
	for _, node := range nodes {
		evaluation := policydomain.Evaluate(policy, node, resources, labels)
		if evaluation.Eligible {
			result = append(result, Candidate{Node: node, Evaluation: evaluation})
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Evaluation.Score == result[j].Evaluation.Score {
			return result[i].Node.ID < result[j].Node.ID
		}
		return result[i].Evaluation.Score > result[j].Evaluation.Score
	})
	return result
}
