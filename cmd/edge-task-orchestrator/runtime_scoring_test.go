package main

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestShutdownCancelsSchedulerWithinDeadline(t *testing.T) {
	cancelRequested := make(chan struct{})
	allowExit := make(chan struct{})
	done := make(chan struct{})
	var calls atomic.Int32
	r := &runtime{
		schedulerCancel: func() {
			calls.Add(1)
			close(cancelRequested)
		},
		schedulerDone: done,
	}
	go func() {
		<-cancelRequested
		<-allowExit
		close(done)
	}()

	returned := make(chan error, 1)
	go func() { returned <- r.stop(context.Background()) }()
	<-cancelRequested
	select {
	case err := <-returned:
		t.Fatalf("stop returned before scheduler exit: %v", err)
	default:
	}
	close(allowExit)
	if err := <-returned; err != nil {
		t.Fatalf("stop returned error: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("cancel calls = %d, want 1", calls.Load())
	}
}

func TestRuntimeStopIsIdempotent(t *testing.T) {
	done := make(chan struct{})
	close(done)
	var calls atomic.Int32
	r := &runtime{schedulerCancel: func() { calls.Add(1) }, schedulerDone: done}

	if err := r.stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("cancel calls = %d, want 1", calls.Load())
	}
}

func TestShutdownTimeoutBoundsInflightWork(t *testing.T) {
	r := &runtime{schedulerCancel: func() {}, schedulerDone: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := r.stop(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stop error = %v, want context deadline", err)
	}
}

func TestCanceledRuntimeLeavesNoSchedulerLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go runScheduler(ctx, done, func(runCtx context.Context) error {
		<-runCtx.Done()
		return runCtx.Err()
	}, nil)
	cancel()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("scheduler completion was not reported after cancellation")
	}

	r := &runtime{schedulerCancel: func() {}, schedulerDone: done}
	server := &http.Server{}
	if err := r.shutdown(context.Background(), server); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}
}
