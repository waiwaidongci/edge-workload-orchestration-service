package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Registry struct {
	requests  atomic.Uint64
	errors    atomic.Uint64
	scheduled atomic.Uint64
	completed atomic.Uint64
}

func New() *Registry           { return &Registry{} }
func (r *Registry) Request()   { r.requests.Add(1) }
func (r *Registry) Error()     { r.errors.Add(1) }
func (r *Registry) Scheduled() { r.scheduled.Add(1) }
func (r *Registry) Completed() { r.completed.Add(1) }
func (r *Registry) Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprintf(w, "# HELP edge_http_requests_total Total HTTP requests.\n# TYPE edge_http_requests_total counter\nedge_http_requests_total %d\n# HELP edge_http_errors_total Total HTTP errors.\n# TYPE edge_http_errors_total counter\nedge_http_errors_total %d\n# TYPE edge_tasks_scheduled_total counter\nedge_tasks_scheduled_total %d\n# TYPE edge_tasks_completed_total counter\nedge_tasks_completed_total %d\n", r.requests.Load(), r.errors.Load(), r.scheduled.Load(), r.completed.Load())
}
