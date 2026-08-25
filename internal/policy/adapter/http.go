package adapter

import (
	"errors"
	"net/http"

	"github.com/example/edge-task-orchestrator/internal/platform/httpapi"
	"github.com/example/edge-task-orchestrator/internal/policy/application"
	policydomain "github.com/example/edge-task-orchestrator/internal/policy/domain"
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
		httpapi.WriteError(w, 422, "policy_invalid", err)
		return
	}
	httpapi.WriteJSON(w, 201, item)
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		httpapi.WriteError(w, 500, "policy_list_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"items": items, "count": len(items)})
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("policyID"))
	if errors.Is(err, policydomain.ErrNotFound) {
		httpapi.WriteError(w, 404, "policy_not_found", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, 500, "policy_get_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, item)
}
func (h *Handler) SetEnabled(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Enabled bool `json:"enabled"`
	}
	if err := httpapi.DecodeJSON(r, &input); err != nil {
		httpapi.WriteError(w, 400, "invalid_request", err)
		return
	}
	item, err := h.service.SetEnabled(r.Context(), r.PathValue("policyID"), input.Enabled)
	if errors.Is(err, policydomain.ErrNotFound) {
		httpapi.WriteError(w, 404, "policy_not_found", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, 500, "policy_update_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, item)
}
