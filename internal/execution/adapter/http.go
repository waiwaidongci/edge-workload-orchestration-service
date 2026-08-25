package adapter

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/example/edge-task-orchestrator/internal/execution/application"
	executiondomain "github.com/example/edge-task-orchestrator/internal/execution/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/httpapi"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	var command application.SubmitCommand
	if err := httpapi.DecodeJSON(r, &command); err != nil {
		httpapi.WriteError(w, 400, "invalid_request", err)
		return
	}
	item, err := h.service.Submit(r.Context(), command)
	if err != nil {
		httpapi.WriteError(w, 422, "task_invalid", err)
		return
	}
	httpapi.WriteJSON(w, 202, item)
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("executionID"))
	if errors.Is(err, executiondomain.ErrNotFound) {
		httpapi.WriteError(w, 404, "execution_not_found", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, 500, "execution_get_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, item)
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	filter := executiondomain.Filter{Status: executiondomain.Status(r.URL.Query().Get("status")), NodeID: r.URL.Query().Get("node_id"), TemplateID: r.URL.Query().Get("template_id"), Limit: limit}
	items, err := h.service.List(r.Context(), filter)
	if err != nil {
		httpapi.WriteError(w, 500, "execution_list_failed", err)
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"items": items, "count": len(items)})
}
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Cancel(r.Context(), r.PathValue("executionID"))
	if errors.Is(err, executiondomain.ErrNotFound) {
		httpapi.WriteError(w, 404, "execution_not_found", err)
		return
	}
	if errors.Is(err, executiondomain.ErrConflict) {
		httpapi.WriteError(w, 409, "state_conflict", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, 500, "execution_cancel_failed", err)
		return
	}
	httpapi.WriteJSON(w, 202, item)
}
func (h *Handler) Receipt(w http.ResponseWriter, r *http.Request) {
	var command application.ReceiptCommand
	if err := httpapi.DecodeJSON(r, &command); err != nil {
		httpapi.WriteError(w, 400, "invalid_request", err)
		return
	}
	item, err := h.service.Receipt(r.Context(), r.PathValue("executionID"), r.PathValue("nodeID"), command)
	if errors.Is(err, executiondomain.ErrNotFound) {
		httpapi.WriteError(w, 404, "execution_not_found", err)
		return
	}
	if errors.Is(err, executiondomain.ErrConflict) {
		httpapi.WriteError(w, 409, "state_conflict", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, 422, "receipt_invalid", err)
		return
	}
	httpapi.WriteJSON(w, 200, item)
}
