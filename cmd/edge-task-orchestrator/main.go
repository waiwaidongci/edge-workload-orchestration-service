package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	capabilityadapter "github.com/example/edge-task-orchestrator/internal/capability/adapter"
	capabilityapplication "github.com/example/edge-task-orchestrator/internal/capability/application"
	capabilityinfrastructure "github.com/example/edge-task-orchestrator/internal/capability/infrastructure"
	executionadapter "github.com/example/edge-task-orchestrator/internal/execution/adapter"
	executionapplication "github.com/example/edge-task-orchestrator/internal/execution/application"
	executioninfrastructure "github.com/example/edge-task-orchestrator/internal/execution/infrastructure"
	deadletteradapter "github.com/example/edge-task-orchestrator/internal/failure/adapter"
	deadletterapplication "github.com/example/edge-task-orchestrator/internal/failure/application"
	deadletterinfrastructure "github.com/example/edge-task-orchestrator/internal/failure/infrastructure"
	heartbeatadapter "github.com/example/edge-task-orchestrator/internal/heartbeat/adapter"
	heartbeatapplication "github.com/example/edge-task-orchestrator/internal/heartbeat/application"
	heartbeatinfrastructure "github.com/example/edge-task-orchestrator/internal/heartbeat/infrastructure"
	nodeadapter "github.com/example/edge-task-orchestrator/internal/node/adapter"
	nodeapplication "github.com/example/edge-task-orchestrator/internal/node/application"
	nodeinfrastructure "github.com/example/edge-task-orchestrator/internal/node/infrastructure"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
	"github.com/example/edge-task-orchestrator/internal/platform/config"
	"github.com/example/edge-task-orchestrator/internal/platform/httpapi"
	"github.com/example/edge-task-orchestrator/internal/platform/logging"
	"github.com/example/edge-task-orchestrator/internal/platform/metrics"
	policyadapter "github.com/example/edge-task-orchestrator/internal/policy/adapter"
	policyapplication "github.com/example/edge-task-orchestrator/internal/policy/application"
	policyinfrastructure "github.com/example/edge-task-orchestrator/internal/policy/infrastructure"
	scheduleradapter "github.com/example/edge-task-orchestrator/internal/scheduler/adapter"
	schedulerapplication "github.com/example/edge-task-orchestrator/internal/scheduler/application"
	templateadapter "github.com/example/edge-task-orchestrator/internal/template/adapter"
	templateapplication "github.com/example/edge-task-orchestrator/internal/template/application"
	templateinfrastructure "github.com/example/edge-task-orchestrator/internal/template/infrastructure"
	"github.com/example/edge-task-orchestrator/internal/transport"
)

//go:embed web/*
var dashboardFiles embed.FS

func main() {
	logger := logging.New()
	configPath := os.Getenv("EDGE_CONFIG")
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}
	application, stop, err := buildApplication(cfg, logger)
	if err != nil {
		logger.Error("build application", "error", err)
		os.Exit(1)
	}
	defer stop()

	server := &http.Server{Addr: cfg.HTTP.Address, Handler: application.handler, ReadTimeout: cfg.HTTP.ReadTimeout, WriteTimeout: cfg.HTTP.WriteTimeout, IdleTimeout: 60 * time.Second}
	shutdownContext, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-shutdownContext.Done()
		ctx, release := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
		defer release()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("HTTP shutdown", "error", err)
		}
	}()
	logger.Info("edge task orchestrator started", "address", cfg.HTTP.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP server stopped", "error", err)
		os.Exit(1)
	}
}

type runtime struct {
	handler         http.Handler
	schedulerCancel context.CancelFunc
}

func (r *runtime) stop() {
	if r.schedulerCancel != nil {
		r.schedulerCancel()
	}
}

func buildApplication(cfg config.Config, logger *slog.Logger) (*runtime, func(), error) {
	clk := clock.Real{}
	nodes := nodeinfrastructure.NewMemoryRepository()
	capabilities := capabilityinfrastructure.NewMemoryRepository()
	templates := templateinfrastructure.NewMemoryRepository()
	policies := policyinfrastructure.NewMemoryRepository()
	executions := executioninfrastructure.NewMemoryRepository()
	heartbeats := heartbeatinfrastructure.NewMemoryRepository()
	presence := heartbeatinfrastructure.NewMemoryPresence()
	deadLetters := deadletterinfrastructure.NewMemoryRepository()
	dispatcher := transport.NewInMemoryDispatcher()
	nodeService := nodeapplication.NewService(nodes, clk)
	capabilityService := capabilityapplication.NewService(capabilities, nodes, clk)
	templateService := templateapplication.NewService(templates, clk)
	policyService := policyapplication.NewService(policies, clk)
	heartbeatService := heartbeatapplication.NewService(nodes, heartbeats, presence, clk, cfg.Scheduler.HeartbeatTimeout)
	deadLetterService := deadletterapplication.NewService(deadLetters, clk)
	executionService := executionapplication.NewService(executions, templates, policies, clk)
	scheduler := schedulerapplication.NewService(executions, nodes, policies, deadLetters, heartbeatService, dispatcher, clk, schedulerapplication.Config{Interval: cfg.Scheduler.Interval, LeaseDuration: cfg.Scheduler.LeaseDuration, HeartbeatTimeout: cfg.Scheduler.HeartbeatTimeout, BaseBackoff: cfg.Scheduler.BaseBackoff, MaxBatch: 32}, logger)
	nodeHandler := nodeadapter.NewHandler(nodeService)
	capabilityHandler := capabilityadapter.NewHandler(capabilityService)
	templateHandler := templateadapter.NewHandler(templateService)
	policyHandler := policyadapter.NewHandler(policyService)
	heartbeatHandler := heartbeatadapter.NewHandler(heartbeatService)
	executionHandler := executionadapter.NewHandler(executionService)
	deadLetterHandler := deadletteradapter.NewHandler(deadLetterService)
	schedulerHandler := scheduleradapter.NewHandler(scheduler)
	registry := metrics.New()
	mux := http.NewServeMux()
	registerRoutes(mux, nodeHandler, capabilityHandler, templateHandler, policyHandler, heartbeatHandler, executionHandler, deadLetterHandler, schedulerHandler, registry)
	limiter := httpapi.NewLimiter(cfg.HTTP.RatePerSecond)
	handler := httpapi.Recover(logger, httpapi.RequestID(httpapi.AccessLog(logger, httpapi.CORS(httpapi.BodyLimit(cfg.HTTP.MaxBodyBytes, httpapi.Timeout(15*time.Second, limiter.Middleware(mux)))))))
	schedulerContext, schedulerCancel := context.WithCancel(context.Background())
	go func() {
		if err := scheduler.Run(schedulerContext); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("scheduler stopped", "error", err)
		}
	}()
	return &runtime{handler: handler, schedulerCancel: schedulerCancel}, func() { schedulerCancel() }, nil
}

func registerRoutes(mux *http.ServeMux, nodes *nodeadapter.Handler, capabilities *capabilityadapter.Handler, templates *templateadapter.Handler, policies *policyadapter.Handler, heartbeats *heartbeatadapter.Handler, executions *executionadapter.Handler, deadLetters *deadletteradapter.Handler, scheduler *scheduleradapter.Handler, registry *metrics.Registry) {
	dashboardRoot, err := fs.Sub(dashboardFiles, "web")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /ui/", http.StripPrefix("/ui/", http.FileServer(http.FS(dashboardRoot))))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui/", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteJSON(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /metrics", registry.Handler)
	mux.HandleFunc("POST /api/v1/nodes", nodes.Register)
	mux.HandleFunc("GET /api/v1/nodes", nodes.List)
	mux.HandleFunc("GET /api/v1/nodes/{nodeID}", nodes.Get)
	mux.HandleFunc("PUT /api/v1/nodes/{nodeID}/capabilities/{name}", capabilities.Upsert)
	mux.HandleFunc("GET /api/v1/nodes/{nodeID}/capabilities", capabilities.List)
	mux.HandleFunc("DELETE /api/v1/nodes/{nodeID}/capabilities/{name}", capabilities.Delete)
	mux.HandleFunc("POST /api/v1/templates", templates.Create)
	mux.HandleFunc("GET /api/v1/templates", templates.List)
	mux.HandleFunc("GET /api/v1/templates/{templateID}", templates.Get)
	mux.HandleFunc("POST /api/v1/templates/{templateID}/versions", templates.AddVersion)
	mux.HandleFunc("POST /api/v1/policies", policies.Create)
	mux.HandleFunc("GET /api/v1/policies", policies.List)
	mux.HandleFunc("GET /api/v1/policies/{policyID}", policies.Get)
	mux.HandleFunc("PATCH /api/v1/policies/{policyID}/enabled", policies.SetEnabled)
	mux.HandleFunc("POST /api/v1/nodes/{nodeID}/heartbeat", heartbeats.Beat)
	mux.HandleFunc("GET /api/v1/nodes/{nodeID}/heartbeats", heartbeats.Recent)
	mux.HandleFunc("POST /api/v1/tasks", executions.Submit)
	mux.HandleFunc("GET /api/v1/tasks", executions.List)
	mux.HandleFunc("GET /api/v1/tasks/{executionID}", executions.Get)
	mux.HandleFunc("POST /api/v1/tasks/{executionID}/cancel", executions.Cancel)
	mux.HandleFunc("POST /api/v1/tasks/{executionID}/nodes/{nodeID}/receipt", executions.Receipt)
	mux.HandleFunc("POST /api/v1/scheduler/simulate", scheduler.Simulate)
	mux.HandleFunc("GET /api/v1/dead-letters", deadLetters.List)
	mux.HandleFunc("POST /api/v1/dead-letters/{deadLetterID}/resolve", deadLetters.Resolve)
}
