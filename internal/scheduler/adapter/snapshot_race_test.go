package adapter

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	executiondomain "github.com/example/edge-task-orchestrator/internal/execution/domain"
	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
	nodeinfra "github.com/example/edge-task-orchestrator/internal/node/infrastructure"
	policydomain "github.com/example/edge-task-orchestrator/internal/policy/domain"
	policyinfra "github.com/example/edge-task-orchestrator/internal/policy/infrastructure"
	schedulerapplication "github.com/example/edge-task-orchestrator/internal/scheduler/application"
	"github.com/example/edge-task-orchestrator/internal/transport"
)

func TestConcurrentSimulationSnapshotsDoNotRace(t *testing.T) {
	handler, _ := newSimulationFixture(t)
	start := make(chan struct{})
	var ready sync.WaitGroup
	var workers sync.WaitGroup
	ready.Add(2)
	workers.Add(2)

	for participant := 0; participant < 2; participant++ {
		go func(participant int) {
			defer workers.Done()
			ready.Done()
			<-start
			for round := 0; round < 10; round++ {
				body := bytes.NewBufferString(`{"policy_id":"policy-a","resources":{"cpu_millis":100,"memory_mb":128},"labels":{"pool":"blue"}}`)
				req := httptest.NewRequest("POST", "/scheduler/simulate", body)
				recorder := httptest.NewRecorder()
				handler.Simulate(recorder, req)
				if recorder.Code != 200 {
					t.Errorf("participant %d round %d: status=%d body=%s", participant, round, recorder.Code, recorder.Body.String())
					return
				}
			}
		}(participant)
	}
	ready.Wait()
	close(start)
	workers.Wait()
}

func TestDispatcherMessagesAreDeepSnapshots(t *testing.T) {
	dispatcher := transport.NewInMemoryDispatcher()
	item := &executiondomain.Execution{ID: "execution-a", NodeID: "node-a", Attempt: 1, UpdatedAt: time.Unix(10, 0)}
	if err := dispatcher.Dispatch(context.Background(), item); err != nil {
		t.Fatal(err)
	}

	messages := dispatcher.Messages()
	if len(messages) != 1 {
		t.Fatalf("messages=%d, want 1", len(messages))
	}
	messages[0].NodeID = "caller-mutated"
	if got := dispatcher.Messages()[0].NodeID; got != "node-a" {
		t.Fatalf("stored message was changed through returned snapshot: %q", got)
	}
}

func TestSimulationReasonsRemainIsolated(t *testing.T) {
	_, service := newSimulationFixture(t)
	command := schedulerapplication.SimulateCommand{
		PolicyID:  "policy-a",
		Resources: nodedomain.Resources{CPUMillis: 100, MemoryMB: 128},
		Labels:    map[string]string{"pool": "blue"},
	}

	first, err := service.Simulate(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Simulate(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) == 0 || len(second) == 0 || len(first[0].Reasons) == 0 || len(second[0].Reasons) == 0 {
		t.Fatalf("expected candidates with reasons: first=%+v second=%+v", first, second)
	}

	want := first[0].Reasons[0]
	second[0].Reasons[0] = "caller-mutated"
	if first[0].Reasons[0] != want {
		t.Fatalf("first simulation changed after mutating second: got %q want %q", first[0].Reasons[0], want)
	}
}

func TestDispatchAndReadPreserveEveryMessage(t *testing.T) {
	dispatcher := transport.NewInMemoryDispatcher()
	start := make(chan struct{})
	var ready sync.WaitGroup
	var workers sync.WaitGroup
	ready.Add(2)
	workers.Add(2)

	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < 128; i++ {
			item := &executiondomain.Execution{ID: fmt.Sprintf("execution-%03d", i), NodeID: "node-a", Attempt: 1, UpdatedAt: time.Unix(int64(i+1), 0)}
			if err := dispatcher.Dispatch(context.Background(), item); err != nil {
				t.Errorf("dispatch %d: %v", i, err)
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < 128; i++ {
			_ = dispatcher.Messages()
		}
	}()
	ready.Wait()
	close(start)
	workers.Wait()

	if got := len(dispatcher.Messages()); got != 128 {
		t.Fatalf("messages=%d, want 128", got)
	}
}

func newSimulationFixture(t *testing.T) (*Handler, *schedulerapplication.Service) {
	t.Helper()
	ctx := context.Background()
	nodes := nodeinfra.NewMemoryRepository()
	policies := policyinfra.NewMemoryRepository()
	now := time.Unix(1_700_000_000, 0).UTC()

	node, err := nodedomain.New("node-a", "edge-a", "region-a", "zone-a", map[string]string{"pool": "blue"}, nodedomain.Resources{CPUMillis: 4000, MemoryMB: 8192}, 16, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodes.Save(ctx, node); err != nil {
		t.Fatal(err)
	}
	policy, err := policydomain.New("policy-a", "latency-policy", "", 10, policydomain.Constraints{
		PreferredRegions: []string{"region-a"},
		AffinityLabels:   map[string]string{"pool": "blue"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := policies.Save(ctx, policy); err != nil {
		t.Fatal(err)
	}

	service := schedulerapplication.NewService(nil, nodes, policies, nil, nil, transport.NewInMemoryDispatcher(), nil, schedulerapplication.Config{}, nil)
	return NewHandler(service), service
}
