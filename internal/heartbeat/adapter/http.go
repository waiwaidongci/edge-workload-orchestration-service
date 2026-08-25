package adapter

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/example/edge-task-orchestrator/internal/heartbeat/application"
	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/httpapi"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }
func (h *Handler) Beat(w http.ResponseWriter, r *http.Request) {
	var command application.BeatCommand
	if err := httpapi.DecodeJSON(r, &command); err != nil {
		httpapi.WriteError(w, 400, "invalid_request", err)
		return
	}
	record, err := h.service.Beat(r.Context(), r.PathValue("nodeID"), command)
	if errors.Is(err, nodedomain.ErrNotFound) {
		httpapi.WriteError(w, 404, "node_not_found", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, 500, "heartbeat_failed", err)
		return
	}
	httpapi.WriteJSON(w, 202, record)
}
func (h *Handler) Recent(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	records, err := h.service.Recent(r.Context(), r.PathValue("nodeID"), limit)
	if err != nil {
		httpapi.WriteError(w, 500, "heartbeat_list_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"items": records, "count": len(records)})
}
