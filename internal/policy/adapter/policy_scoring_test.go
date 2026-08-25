package adapter

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	policyapp "github.com/example/edge-task-orchestrator/internal/policy/application"
	policydomain "github.com/example/edge-task-orchestrator/internal/policy/domain"
	policyinfra "github.com/example/edge-task-orchestrator/internal/policy/infrastructure"
)

func TestEvaluateMissingInputsRejectsWithoutPanic(t *testing.T) {
	result := policydomain.Evaluate(nil, nil, nodedomain.Resources{}, nil)
	if result.Eligible {
		t.Fatalf("missing policy/node was eligible: %#v", result)
	}
}

func TestPolicyConstructorOwnsRegionSlices(t *testing.T) {
	preferred := []string{"us-east"}
	required := []string{"us-east"}
	item, err := policydomain.New("p-1", "policy", "", 1, policydomain.Constraints{PreferredRegions: preferred, RequiredRegions: required}, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	preferred[0] = "changed"
	required[0] = "changed"
	if item.Constraints.PreferredRegions[0] != "us-east" || item.Constraints.RequiredRegions[0] != "us-east" {
		t.Fatalf("policy retained caller slices: %#v", item.Constraints)
	}
}

func TestPolicyRepositoryOwnsRegionSlices(t *testing.T) {
	repo := policyinfra.NewMemoryRepository()
	item, err := policydomain.New("p-1", "policy", "", 1, policydomain.Constraints{PreferredRegions: []string{"us-east"}, RequiredRegions: []string{"us-east"}}, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	got.Constraints.PreferredRegions[0] = "mutated"
	again, err := repo.Get(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Constraints.PreferredRegions[0] != "us-east" || len(again.Constraints.RequiredRegions) != 1 {
		t.Fatalf("repository snapshot was not isolated: %#v", again.Constraints)
	}
}

func TestPolicyGetMissingMapsToNotFound(t *testing.T) {
	service := policyapp.NewService(policyinfra.NewMemoryRepository(), clock.Fixed{Time: time.Unix(100, 0)})
	h := NewHandler(service)
	r := httptest.NewRequest("GET", "/policies/missing", nil)
	r.SetPathValue("policyID", "missing")
	w := httptest.NewRecorder()
	h.Get(w, r)
	if w.Code != 404 {
		t.Fatalf("status=%d, want 404", w.Code)
	}
}
