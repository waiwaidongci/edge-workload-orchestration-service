package adapter

import (
	"net/http"
	"strconv"

	"github.com/example/edge-task-orchestrator/internal/failure/application"
	"github.com/example/edge-task-orchestrator/internal/platform/httpapi"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	includeResolved := r.URL.Query().Get("include_resolved") == "true"
	items, err := h.service.List(r.Context(), includeResolved, limit)
	if err != nil {
		httpapi.WriteError(w, 500, "dead_letter_list_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"items": items, "count": len(items)})
}
func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Note string `json:"note"`
	}
	if err := httpapi.DecodeJSON(r, &input); err != nil {
		httpapi.WriteError(w, 400, "invalid_request", err)
		return
	}
	item, err := h.service.Resolve(r.Context(), r.PathValue("deadLetterID"), input.Note)
	if err != nil {
		httpapi.WriteError(w, 404, "dead_letter_not_found", err)
		return
	}
	httpapi.WriteJSON(w, 200, item)
}
