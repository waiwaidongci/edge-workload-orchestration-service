package adapter

import (
	"errors"
	"net/http"

	"github.com/example/edge-task-orchestrator/internal/node/application"
	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/httpapi"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var command application.RegisterCommand
	if err := httpapi.DecodeJSON(r, &command); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	node, err := h.service.Register(r.Context(), command)
	if err != nil {
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "node_invalid", err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, node)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := nodedomain.Filter{Region: r.URL.Query().Get("region"), Status: nodedomain.Status(r.URL.Query().Get("status")), Labels: httpapi.ParseLabels(r.URL.Query()["label"])}
	nodes, err := h.service.List(r.Context(), filter)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "node_list_failed", err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"items": nodes, "count": len(nodes)})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	node, err := h.service.Get(r.Context(), r.PathValue("nodeID"))
	if errors.Is(err, nodedomain.ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "node_not_found", err)
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "node_get_failed", err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, node)
}
