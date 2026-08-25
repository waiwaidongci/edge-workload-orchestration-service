package main

import (
	"context"
	"net/http"
)

type runtime struct {
	handler         http.Handler
	schedulerCancel context.CancelFunc
	schedulerDone   <-chan struct{}
}

func (r *runtime) stop(context.Context) error {
	if r.schedulerCancel != nil {
		r.schedulerCancel()
	}
	return nil
}

func (r *runtime) shutdown(ctx context.Context, server *http.Server) error {
	return server.Shutdown(ctx)
}
