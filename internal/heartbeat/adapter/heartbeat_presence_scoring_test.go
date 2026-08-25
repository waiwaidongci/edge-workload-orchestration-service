package adapter_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/example/edge-task-orchestrator/internal/heartbeat/application"
	heartbeatdomain "github.com/example/edge-task-orchestrator/internal/heartbeat/domain"
	heartbeatinfra "github.com/example/edge-task-orchestrator/internal/heartbeat/infrastructure"
	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	"github.com/example/edge-task-orchestrator/internal/platform/clock"
)

const presenceRaceIterations = 2000

func runTogether(workers ...func()) {
	start := make(chan struct{})
	var ready sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(len(workers))
	done.Add(len(workers))
	for _, work := range workers {
		work := work
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			work()
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
}

func TestPresenceTouchSerializesConcurrentWriters(t *testing.T) {
	presence := heartbeatinfra.NewMemoryPresence()
	runTogether(
		func() {
			for i := 0; i < presenceRaceIterations; i++ {
				if err := presence.Touch(context.Background(), "node-a", time.Minute); err != nil {
					t.Errorf("touch node: %v", err)
				}
			}
		},
		func() {
			for i := 0; i < presenceRaceIterations; i++ {
				if err := presence.Touch(context.Background(), "node-a", 2*time.Minute); err != nil {
					t.Errorf("touch node: %v", err)
				}
			}
		},
	)
}

func TestPresenceOnlineSynchronizesWithTouch(t *testing.T) {
	presence := heartbeatinfra.NewMemoryPresence()
	if err := presence.Touch(context.Background(), "node-a", time.Minute); err != nil {
		t.Fatalf("seed presence: %v", err)
	}
	runTogether(
		func() {
			for i := 0; i < presenceRaceIterations; i++ {
				_ = presence.Touch(context.Background(), "node-a", time.Minute)
			}
		},
		func() {
			for i := 0; i < presenceRaceIterations; i++ {
				_, _ = presence.Online(context.Background(), "node-a")
			}
		},
	)
}

func TestPresenceRemoveSynchronizesWithTouch(t *testing.T) {
	presence := heartbeatinfra.NewMemoryPresence()
	runTogether(
		func() {
			for i := 0; i < presenceRaceIterations; i++ {
				_ = presence.Touch(context.Background(), "node-a", time.Minute)
			}
		},
		func() {
			for i := 0; i < presenceRaceIterations; i++ {
				_ = presence.Remove(context.Background(), "node-a")
			}
		},
	)
}

type heartbeatNodeRepository struct {
	item *nodedomain.Node
}

func (*heartbeatNodeRepository) Save(context.Context, *nodedomain.Node) error { return nil }

func (r *heartbeatNodeRepository) Get(context.Context, string) (*nodedomain.Node, error) {
	return r.item, nil
}

func (r *heartbeatNodeRepository) List(context.Context, nodedomain.Filter) ([]*nodedomain.Node, error) {
	return []*nodedomain.Node{r.item}, nil
}

func (r *heartbeatNodeRepository) Update(context.Context, *nodedomain.Node) error { return nil }

type heartbeatRepository struct{}

func (*heartbeatRepository) Append(context.Context, heartbeatdomain.Record) error { return nil }
func (*heartbeatRepository) Recent(context.Context, string, time.Time, int) ([]heartbeatdomain.Record, error) {
	return nil, nil
}
func (*heartbeatRepository) AppendStatusChange(context.Context, heartbeatdomain.StatusChange) error {
	return nil
}
func (*heartbeatRepository) StatusChanges(context.Context, string, int) ([]heartbeatdomain.StatusChange, error) {
	return nil, nil
}

type blockingPresence struct {
	touchStarted  chan struct{}
	touchRelease  chan struct{}
	removeStarted chan struct{}
	removeRelease chan struct{}
	err           error
}

func (p *blockingPresence) Touch(context.Context, string, time.Duration) error {
	close(p.touchStarted)
	<-p.touchRelease
	return p.err
}

func (*blockingPresence) Online(context.Context, string) (bool, error) { return true, nil }

func (p *blockingPresence) Remove(context.Context, string) error {
	close(p.removeStarted)
	<-p.removeRelease
	return p.err
}

func TestHeartbeatBeatWaitsForPresenceRefresh(t *testing.T) {
	downstreamErr := errors.New("presence refresh failed")
	now := time.Unix(1_700_000_000, 0).UTC()
	node := &nodedomain.Node{ID: "node-a", Status: nodedomain.StatusOnline, LastHeartbeat: now}
	presence := &blockingPresence{
		touchStarted:  make(chan struct{}),
		touchRelease:  make(chan struct{}),
		removeStarted: make(chan struct{}),
		removeRelease: make(chan struct{}),
		err:           downstreamErr,
	}
	service := application.NewService(&heartbeatNodeRepository{item: node}, &heartbeatRepository{}, presence, clock.Fixed{Time: now}, time.Minute)
	start := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		<-start
		_, err := service.Beat(context.Background(), "node-a", application.BeatCommand{})
		done <- err
	}()
	close(start)
	<-presence.touchStarted
	runtime.Gosched()
	select {
	case err := <-done:
		close(presence.touchRelease)
		t.Fatalf("Beat returned before presence refresh completed: %v", err)
	default:
	}
	close(presence.touchRelease)
	if err := <-done; !errors.Is(err, downstreamErr) {
		t.Fatalf("presence refresh error lost: %v", err)
	}
}

func TestHeartbeatSweepWaitsForPresenceRemoval(t *testing.T) {
	downstreamErr := errors.New("presence removal failed")
	now := time.Unix(1_700_000_000, 0).UTC()
	node := &nodedomain.Node{ID: "node-a", Status: nodedomain.StatusOnline, LastHeartbeat: now.Add(-2 * time.Hour)}
	presence := &blockingPresence{
		touchStarted:  make(chan struct{}),
		touchRelease:  make(chan struct{}),
		removeStarted: make(chan struct{}),
		removeRelease: make(chan struct{}),
		err:           downstreamErr,
	}
	service := application.NewService(&heartbeatNodeRepository{item: node}, &heartbeatRepository{}, presence, clock.Fixed{Time: now}, time.Hour)
	start := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		<-start
		_, err := service.Sweep(context.Background())
		done <- err
	}()
	close(start)
	<-presence.removeStarted
	runtime.Gosched()
	select {
	case err := <-done:
		close(presence.removeRelease)
		t.Fatalf("Sweep returned before presence removal completed: %v", err)
	default:
	}
	close(presence.removeRelease)
	if err := <-done; !errors.Is(err, downstreamErr) {
		t.Fatalf("presence removal error lost: %v", err)
	}
}
