package adapter

import (
	"errors"
	"net/http"

	"github.com/example/edge-task-orchestrator/internal/capability/application"
	capabilitydomain "github.com/example/edge-task-orchestrator/internal/capability/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/httpapi"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	var command application.UpsertCommand
	if err := httpapi.DecodeJSON(r, &command); err != nil {
		httpapi.WriteError(w, 400, "invalid_request", err)
		return
	}
	if command.Name == "" {
		command.Name = r.PathValue("name")
	}
	item, err := h.service.Upsert(r.Context(), r.PathValue("nodeID"), command)
	if err != nil {
		httpapi.WriteError(w, 422, "capability_invalid", err)
		return
	}
	httpapi.WriteJSON(w, 201, item)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), r.PathValue("nodeID"))
	if err != nil {
		httpapi.WriteError(w, 500, "capability_list_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"items": items, "count": len(items)})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.service.Delete(r.Context(), r.PathValue("nodeID"), r.PathValue("name"))
	if errors.Is(err, capabilitydomain.ErrNotFound) {
		httpapi.WriteError(w, 404, "capability_not_found", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, 500, "capability_delete_failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
