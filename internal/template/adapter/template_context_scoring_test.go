package adapter_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	templateadapter "github.com/example/edge-task-orchestrator/internal/template/adapter"
	"github.com/example/edge-task-orchestrator/internal/template/application"
	templatedomain "github.com/example/edge-task-orchestrator/internal/template/domain"
)

type contextRepository struct {
	calls atomic.Int32
	item  *templatedomain.Template
}

func (r *contextRepository) observe(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.calls.Add(1)
	return nil
}

func (r *contextRepository) Save(ctx context.Context, item *templatedomain.Template) error {
	if err := r.observe(ctx); err != nil {
		return err
	}
	r.item = item
	return nil
}

func (r *contextRepository) Get(ctx context.Context, _ string) (*templatedomain.Template, error) {
	if err := r.observe(ctx); err != nil {
		return nil, err
	}
	return r.item, nil
}

func (r *contextRepository) List(ctx context.Context) ([]*templatedomain.Template, error) {
	if err := r.observe(ctx); err != nil {
		return nil, err
	}
	return []*templatedomain.Template{r.item}, nil
}

func (r *contextRepository) Update(ctx context.Context, item *templatedomain.Template) error {
	if err := r.observe(ctx); err != nil {
		return err
	}
	r.item = item
	return nil
}

func canceledRequest(method, target, body string) *http.Request {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	return request.WithContext(ctx)
}

func templateHandler(t *testing.T) (*templateadapter.Handler, *contextRepository) {
	t.Helper()
	now := time.Unix(1_700_000_000, 0).UTC()
	item, err := templatedomain.New("template-a", "camera-process", "", now)
	if err != nil {
		t.Fatalf("new template: %v", err)
	}
	repository := &contextRepository{item: item}
	service := application.NewService(repository, clock.Fixed{Time: now})
	return templateadapter.NewHandler(service), repository
}

func TestTemplateCreatePropagatesCanceledRequest(t *testing.T) {
	handler, repository := templateHandler(t)
	request := canceledRequest(http.MethodPost, "/api/v1/templates", `{"name":"camera-process"}`)
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if repository.calls.Load() != 0 {
		t.Fatalf("repository called after request cancellation: %d", repository.calls.Load())
	}
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d want %d", response.Code, http.StatusUnprocessableEntity)
	}
}

func TestTemplateAddVersionPropagatesCanceledRequest(t *testing.T) {
	handler, repository := templateHandler(t)
	request := canceledRequest(http.MethodPost, "/api/v1/templates/template-a/versions", `{"image":"edge/camera:v2","timeout_seconds":30,"resources":{"cpu_millis":100,"memory_mb":128}}`)
	request.SetPathValue("templateID", "template-a")
	response := httptest.NewRecorder()

	handler.AddVersion(response, request)

	if repository.calls.Load() != 0 {
		t.Fatalf("repository called after request cancellation: %d", repository.calls.Load())
	}
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d want %d", response.Code, http.StatusUnprocessableEntity)
	}
}

func TestTemplateGetPropagatesCanceledRequest(t *testing.T) {
	handler, repository := templateHandler(t)
	request := canceledRequest(http.MethodGet, "/api/v1/templates/template-a", "")
	request.SetPathValue("templateID", "template-a")
	response := httptest.NewRecorder()

	handler.Get(response, request)

	if repository.calls.Load() != 0 {
		t.Fatalf("repository called after request cancellation: %d", repository.calls.Load())
	}
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", response.Code, http.StatusInternalServerError)
	}
}

func TestTemplateListPropagatesCanceledRequest(t *testing.T) {
	handler, repository := templateHandler(t)
	request := canceledRequest(http.MethodGet, "/api/v1/templates", "")
	response := httptest.NewRecorder()

	handler.List(response, request)

	if repository.calls.Load() != 0 {
		t.Fatalf("repository called after request cancellation: %d", repository.calls.Load())
	}
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", response.Code, http.StatusInternalServerError)
	}
}
