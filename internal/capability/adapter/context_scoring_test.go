package adapter

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/edge-task-orchestrator/internal/capability/application"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
)

func TestCanceledCapabilityUpsertReachesNodeRepository(t *testing.T) {
	handler, repository, nodes := canceledCapabilityHandler()
	request := canceledRequest("PUT", "/capabilities/gpu", `{"name":"gpu","version":"1"}`)
	request.SetPathValue("nodeID", "node-a")
	request.SetPathValue("name", "gpu")
	handler.Upsert(httptest.NewRecorder(), request)
	if !errors.Is(nodes.seenGet, context.Canceled) || repository.seenSave != nil {
		t.Fatalf("upsert context: node=%v save=%v", nodes.seenGet, repository.seenSave)
	}
}

func TestCanceledCapabilityListReachesRepository(t *testing.T) {
	handler, repository, _ := canceledCapabilityHandler()
	request := canceledRequest("GET", "/capabilities", "")
	request.SetPathValue("nodeID", "node-a")
	handler.List(httptest.NewRecorder(), request)
	if !errors.Is(repository.seenList, context.Canceled) {
		t.Fatalf("list context = %v", repository.seenList)
	}
}

func TestCanceledCapabilityDeleteReachesRepository(t *testing.T) {
	handler, repository, _ := canceledCapabilityHandler()
	request := canceledRequest("DELETE", "/capabilities/gpu", "")
	request.SetPathValue("nodeID", "node-a")
	request.SetPathValue("name", "gpu")
	handler.Delete(httptest.NewRecorder(), request)
	if !errors.Is(repository.seenDelete, context.Canceled) {
		t.Fatalf("delete context = %v", repository.seenDelete)
	}
}

func TestCapabilityServiceKeepsExpiredDeadline(t *testing.T) {
	repository := &contextCapabilityRepository{}
	service := application.NewService(repository, &contextNodeRepository{}, clock.Fixed{Time: time.Now()})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err := service.List(ctx, "node-a")
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(repository.seenList, context.DeadlineExceeded) {
		t.Fatalf("deadline propagation: err=%v seen=%v", err, repository.seenList)
	}
}
