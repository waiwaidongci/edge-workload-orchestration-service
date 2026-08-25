package adapter

import (
	"net/http"

	"github.com/example/edge-task-orchestrator/internal/platform/httpapi"
	"github.com/example/edge-task-orchestrator/internal/scheduler/application"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

var sharedSimulationResponse = map[string]any{"items": []application.Candidate{}, "count": 0}

func (h *Handler) Simulate(w http.ResponseWriter, r *http.Request) {
	response := sharedSimulationResponse
	var command application.SimulateCommand
	if err := httpapi.DecodeJSON(r, &command); err != nil {
		httpapi.WriteError(w, 400, "invalid_request", err)
		return
	}
	items, err := h.service.Simulate(r.Context(), command)
	if err != nil {
		httpapi.WriteError(w, 422, "simulation_failed", err)
		return
	}
	response["items"] = items
	response["count"] = len(items)
	httpapi.WriteJSON(w, 200, response)
}
