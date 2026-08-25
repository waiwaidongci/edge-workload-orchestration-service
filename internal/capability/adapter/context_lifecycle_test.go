package adapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/edge-task-orchestrator/internal/capability/application"
	capabilitydomain "github.com/example/edge-task-orchestrator/internal/capability/domain"
	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
)

type contextCapabilityRepository struct {
	seenSave   error
	seenList   error
	seenDelete error
}

func (r *contextCapabilityRepository) Save(ctx context.Context, _ *capabilitydomain.Capability) error {
	r.seenSave = ctx.Err()
	return ctx.Err()
}
func (r *contextCapabilityRepository) ListByNode(ctx context.Context, _ string) ([]*capabilitydomain.Capability, error) {
	r.seenList = ctx.Err()
	return nil, ctx.Err()
}
func (*contextCapabilityRepository) Find(context.Context, string, string) (*capabilitydomain.Capability, error) {
	return nil, capabilitydomain.ErrNotFound
}
func (r *contextCapabilityRepository) Delete(ctx context.Context, _, _ string) error {
	r.seenDelete = ctx.Err()
	return ctx.Err()
}

type contextNodeRepository struct{ seenGet error }

func (*contextNodeRepository) Save(context.Context, *nodedomain.Node) error { return nil }
func (r *contextNodeRepository) Get(ctx context.Context, _ string) (*nodedomain.Node, error) {
	r.seenGet = ctx.Err()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &nodedomain.Node{ID: "node-a"}, nil
}
func (*contextNodeRepository) List(context.Context, nodedomain.Filter) ([]*nodedomain.Node, error) {
	return nil, nil
}
func (*contextNodeRepository) Update(context.Context, *nodedomain.Node) error { return nil }

func canceledCapabilityHandler() (*Handler, *contextCapabilityRepository, *contextNodeRepository) {
	repository := &contextCapabilityRepository{}
	nodes := &contextNodeRepository{}
	service := application.NewService(repository, nodes, clock.Fixed{Time: time.Now()})
	return NewHandler(service), repository, nodes
}

func canceledRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	return request.WithContext(ctx)
}

func TestCanceledCapabilityRequestsKeepContext(t *testing.T) {
	t.Run("upsert", func(t *testing.T) {
		handler, repository, nodes := canceledCapabilityHandler()
		request := canceledRequest("PUT", "/capabilities/gpu", `{"name":"gpu","version":"1"}`)
		request.SetPathValue("nodeID", "node-a")
		request.SetPathValue("name", "gpu")
		handler.Upsert(httptest.NewRecorder(), request)
		if !errors.Is(nodes.seenGet, context.Canceled) || repository.seenSave != nil {
			t.Fatalf("upsert detached context: node=%v save=%v", nodes.seenGet, repository.seenSave)
		}
	})

	t.Run("list", func(t *testing.T) {
		handler, repository, _ := canceledCapabilityHandler()
		request := canceledRequest("GET", "/capabilities", "")
		request.SetPathValue("nodeID", "node-a")
		handler.List(httptest.NewRecorder(), request)
		if !errors.Is(repository.seenList, context.Canceled) {
			t.Fatalf("list detached context: %v", repository.seenList)
		}
	})

	t.Run("delete", func(t *testing.T) {
		handler, repository, _ := canceledCapabilityHandler()
		request := canceledRequest("DELETE", "/capabilities/gpu", "")
		request.SetPathValue("nodeID", "node-a")
		request.SetPathValue("name", "gpu")
		handler.Delete(httptest.NewRecorder(), request)
		if !errors.Is(repository.seenDelete, context.Canceled) {
			t.Fatalf("delete detached context: %v", repository.seenDelete)
		}
	})
}
