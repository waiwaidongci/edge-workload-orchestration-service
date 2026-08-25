package adapter

import (
	"context"
	"sync"
	"testing"
	"time"

	templatedomain "github.com/example/edge-task-orchestrator/internal/template/domain"
	templateinfra "github.com/example/edge-task-orchestrator/internal/template/infrastructure"
)

func scoringTemplate(t *testing.T) *templatedomain.Template {
	t.Helper()
	template, err := templatedomain.New("template-1", "edge", "", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := template.AddVersion(templatedomain.Version{Image: "edge:v1", Command: []string{"run", "--safe"}, Environment: map[string]string{"MODE": "safe"}, RequiredCapabilities: []string{"gpu", "gpu"}, TimeoutSeconds: 10}, time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	return template
}

func TestTemplateVersionDoesNotMutateInputSlices(t *testing.T) {
	template, err := templatedomain.New("template-1", "edge", "", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	command := []string{"run", "--safe"}
	capabilities := []string{"gpu", "gpu"}
	environment := map[string]string{"MODE": "safe"}
	if _, err := template.AddVersion(templatedomain.Version{Image: "edge:v1", Command: command, Environment: environment, RequiredCapabilities: capabilities, TimeoutSeconds: 10}, time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	command[0] = "mutated"
	capabilities[0] = "mutated"
	environment["MODE"] = "mutated"
	version, err := template.Version(1)
	if err != nil {
		t.Fatal(err)
	}
	if version.Command[0] != "run" || version.RequiredCapabilities[0] != "gpu" || version.Environment["MODE"] != "safe" {
		t.Fatalf("input mutation polluted version: %#v", version)
	}
	direct := &templatedomain.Template{ID: "direct", LatestVersion: 1, Versions: []templatedomain.Version{{Number: 1, Image: "edge:v1", Command: []string{"run"}, Environment: map[string]string{"MODE": "safe"}}}}
	returned, err := direct.Version(1)
	if err != nil {
		t.Fatal(err)
	}
	returned.Command[0] = "returned-mutated"
	returned.Environment["MODE"] = "returned-mutated"
	if direct.Versions[0].Command[0] != "run" || direct.Versions[0].Environment["MODE"] != "safe" {
		t.Fatalf("Version returned shared mutable data: %#v", direct.Versions[0])
	}
}

func TestTemplateRepositoryReturnsDeepSnapshots(t *testing.T) {
	repository := templateinfra.NewMemoryRepository()
	original := scoringTemplate(t)
	if err := repository.Save(context.Background(), original); err != nil {
		t.Fatal(err)
	}
	first, err := repository.Get(context.Background(), original.ID)
	if err != nil {
		t.Fatal(err)
	}
	first.Versions[0].Environment["MODE"] = "mutated"
	first.Versions[0].Command[0] = "mutated"
	second, err := repository.Get(context.Background(), original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.Versions[0].Environment["MODE"] != "safe" || second.Versions[0].Command[0] != "run" {
		t.Fatalf("repository snapshot was polluted: %#v", second.Versions[0])
	}
}

func TestTemplateEnvironmentMapIsIsolated(t *testing.T) {
	repository := templateinfra.NewMemoryRepository()
	original := scoringTemplate(t)
	if err := repository.Save(context.Background(), original); err != nil {
		t.Fatal(err)
	}
	listed, err := repository.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	listed[0].Versions[0].Environment["NEW"] = "value"
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(2)
	go func() { defer wait.Done(); <-start; listed[0].Versions[0].Environment["RACE"] = "writer" }()
	go func() { defer wait.Done(); <-start; for i := 0; i < 20; i++ { _, _ = repository.Get(context.Background(), original.ID) } }()
	close(start)
	wait.Wait()
	latest, err := repository.Get(context.Background(), original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := latest.Versions[0].Environment["NEW"]; ok {
		t.Fatalf("environment map leaked through list snapshot: %#v", latest.Versions[0].Environment)
	}
}

func TestAppendingNextVersionCannotRewriteHistory(t *testing.T) {
	repository := templateinfra.NewMemoryRepository()
	original := scoringTemplate(t)
	if err := repository.Save(context.Background(), original); err != nil {
		t.Fatal(err)
	}
	loaded, err := repository.Get(context.Background(), original.ID)
	if err != nil {
		t.Fatal(err)
	}
	before, err := repository.Get(context.Background(), original.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.Versions[0].Environment["MODE"] = "history-mutated"
	if _, err := loaded.AddVersion(templatedomain.Version{Image: "edge:v2", Command: []string{"run", "--fast"}, Environment: map[string]string{"MODE": "fast"}, TimeoutSeconds: 10}, time.Unix(3, 0)); err != nil {
		t.Fatal(err)
	}
	first, err := before.Version(1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Image != "edge:v1" || first.Environment["MODE"] != "safe" {
		t.Fatalf("prior snapshot was rewritten by next version: %#v", first)
	}
}
