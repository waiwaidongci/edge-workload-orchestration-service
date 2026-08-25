package adapter

import (
	"errors"
	"net/http"

	"github.com/example/edge-task-orchestrator/internal/platform/httpapi"
	"github.com/example/edge-task-orchestrator/internal/template/application"
	templatedomain "github.com/example/edge-task-orchestrator/internal/template/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var command application.CreateCommand
	if err := httpapi.DecodeJSON(r, &command); err != nil {
		httpapi.WriteError(w, 400, "invalid_request", err)
		return
	}
	item, err := h.service.Create(r.Context(), command)
	if err != nil {
		httpapi.WriteError(w, 422, "template_invalid", err)
		return
	}
	httpapi.WriteJSON(w, 201, item)
}
func (h *Handler) AddVersion(w http.ResponseWriter, r *http.Request) {
	var version templatedomain.Version
	if err := httpapi.DecodeJSON(r, &version); err != nil {
		httpapi.WriteError(w, 400, "invalid_request", err)
		return
	}
	item, err := h.service.AddVersion(r.Context(), r.PathValue("templateID"), version)
	if errors.Is(err, templatedomain.ErrNotFound) {
		httpapi.WriteError(w, 404, "template_not_found", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, 422, "version_invalid", err)
		return
	}
	httpapi.WriteJSON(w, 201, item)
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		httpapi.WriteError(w, 500, "template_list_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"items": items, "count": len(items)})
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("templateID"))
	if errors.Is(err, templatedomain.ErrNotFound) {
		httpapi.WriteError(w, 404, "template_not_found", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, 500, "template_get_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, item)
}
