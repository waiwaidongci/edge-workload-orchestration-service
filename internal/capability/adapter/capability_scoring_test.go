package adapter_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	capabilitydomain "github.com/example/edge-task-orchestrator/internal/capability/domain"
	capabilityinfra "github.com/example/edge-task-orchestrator/internal/capability/infrastructure"
)

const raceIterations = 2000

func capabilityFixture(name string) *capabilitydomain.Capability {
	now := time.Unix(1_700_000_000, 0).UTC()
	return &capabilitydomain.Capability{
		ID:         "cap-" + name,
		NodeID:     "node-a",
		Name:       name,
		Version:    "v1",
		Attributes: map[string]string{"profile": "stable"},
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestCapabilityNewOwnsAttributesDuringConcurrentUse(t *testing.T) {
	attributes := map[string]string{"profile": "stable"}
	item, err := capabilitydomain.New("cap-new", "node-a", "camera", "v1", attributes, true, time.Now())
	if err != nil {
		t.Fatalf("new capability: %v", err)
	}

	start := make(chan struct{})
	var ready sync.WaitGroup
	var workers sync.WaitGroup
	ready.Add(2)
	workers.Add(2)
	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < raceIterations; i++ {
			attributes["profile"] = strconv.Itoa(i)
		}
	}()
	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < raceIterations; i++ {
			_ = item.Attributes["profile"]
		}
	}()
	ready.Wait()
	close(start)
	workers.Wait()
	if item.Attributes["profile"] != "stable" {
		t.Fatalf("created capability changed with caller map: %q", item.Attributes["profile"])
	}
}

func TestCapabilitySaveOwnsSubmittedEntityDuringConcurrentUse(t *testing.T) {
	repository := capabilityinfra.NewMemoryRepository()
	item := capabilityFixture("save")
	if err := repository.Save(context.Background(), item); err != nil {
		t.Fatalf("save capability: %v", err)
	}

	start := make(chan struct{})
	var ready sync.WaitGroup
	var workers sync.WaitGroup
	ready.Add(2)
	workers.Add(2)
	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < raceIterations; i++ {
			item.Version = strconv.Itoa(i)
		}
	}()
	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < raceIterations; i++ {
			stored, err := repository.Find(context.Background(), item.NodeID, item.Name)
			if err != nil {
				t.Errorf("find capability: %v", err)
				continue
			}
			_ = stored.Version
		}
	}()
	ready.Wait()
	close(start)
	workers.Wait()
	stored, err := repository.Find(context.Background(), item.NodeID, item.Name)
	if err != nil {
		t.Fatalf("find saved capability: %v", err)
	}
	if stored.Version != "v1" {
		t.Fatalf("saved capability changed with submitted entity: %q", stored.Version)
	}
}

func TestCapabilityFindReturnsOwnedSnapshotDuringConcurrentUse(t *testing.T) {
	repository := capabilityinfra.NewMemoryRepository()
	item := capabilityFixture("find")
	if err := repository.Save(context.Background(), item); err != nil {
		t.Fatalf("save capability: %v", err)
	}
	result, err := repository.Find(context.Background(), item.NodeID, item.Name)
	if err != nil {
		t.Fatalf("find capability: %v", err)
	}

	start := make(chan struct{})
	var ready sync.WaitGroup
	var workers sync.WaitGroup
	ready.Add(2)
	workers.Add(2)
	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < raceIterations; i++ {
			result.Version = strconv.Itoa(i)
		}
	}()
	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < raceIterations; i++ {
			snapshot, findErr := repository.Find(context.Background(), item.NodeID, item.Name)
			if findErr != nil {
				t.Errorf("find snapshot: %v", findErr)
				continue
			}
			_ = snapshot.Version
		}
	}()
	ready.Wait()
	close(start)
	workers.Wait()
	stored, err := repository.Find(context.Background(), item.NodeID, item.Name)
	if err != nil {
		t.Fatalf("find stored capability: %v", err)
	}
	if stored.Version != "v1" {
		t.Fatalf("stored capability changed through Find result: %q", stored.Version)
	}
}

func TestCapabilityListReturnsOwnedSnapshotsDuringConcurrentUse(t *testing.T) {
	repository := capabilityinfra.NewMemoryRepository()
	item := capabilityFixture("list")
	if err := repository.Save(context.Background(), item); err != nil {
		t.Fatalf("save capability: %v", err)
	}
	results, err := repository.ListByNode(context.Background(), item.NodeID)
	if err != nil {
		t.Fatalf("list capabilities: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("list count: got %d want 1", len(results))
	}

	start := make(chan struct{})
	var ready sync.WaitGroup
	var workers sync.WaitGroup
	ready.Add(2)
	workers.Add(2)
	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < raceIterations; i++ {
			results[0].Version = strconv.Itoa(i)
		}
	}()
	go func() {
		defer workers.Done()
		ready.Done()
		<-start
		for i := 0; i < raceIterations; i++ {
			snapshots, listErr := repository.ListByNode(context.Background(), item.NodeID)
			if listErr != nil {
				t.Errorf("list snapshots: %v", listErr)
				continue
			}
			if len(snapshots) == 1 {
				_ = snapshots[0].Version
			}
		}
	}()
	ready.Wait()
	close(start)
	workers.Wait()
	stored, err := repository.Find(context.Background(), item.NodeID, item.Name)
	if err != nil {
		t.Fatalf("find stored capability: %v", err)
	}
	if stored.Version != "v1" {
		t.Fatalf("stored capability changed through List result: %q", stored.Version)
	}
}
