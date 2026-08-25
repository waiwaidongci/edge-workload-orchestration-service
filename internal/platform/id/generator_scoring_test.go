package id

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestZeroValueGeneratorInitializesState(t *testing.T) {
	var g Generator
	value, err := g.Next("task")
	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if !strings.HasPrefix(value, "task_") {
		t.Fatalf("identifier = %q, want task prefix", value)
	}
}

type shortReader struct{ used bool }

func (r *shortReader) Read(p []byte) (int, error) {
	if r.used {
		return 0, io.EOF
	}
	r.used = true
	copy(p, []byte{1, 2, 3})
	return 3, nil
}

func TestShortEntropyReadReturnsError(t *testing.T) {
	g := NewGenerator(&shortReader{}, func() time.Time { return time.UnixMilli(100) })
	if _, err := g.Next("task"); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Next error = %v, want io.ErrUnexpectedEOF", err)
	}
}

type lockedEntropy struct {
	mu sync.Mutex
	n  byte
}

func (r *lockedEntropy) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range p {
		p[i] = r.n
		r.n++
	}
	return len(p), nil
}

func TestConcurrentGeneratorHasNoRaceOrDuplicates(t *testing.T) {
	g := NewGenerator(&lockedEntropy{}, func() time.Time { return time.UnixMilli(100) })
	start := make(chan struct{})
	results := make(chan string, 32)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			value, err := g.Next("task")
			if err != nil {
				results <- "error:" + err.Error()
				return
			}
			results <- value
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	seen := make(map[string]struct{}, 32)
	entropySeen := make(map[string]struct{}, 32)
	for value := range results {
		if strings.HasPrefix(value, "error:") {
			t.Fatal(value)
		}
		if _, exists := seen[value]; exists {
			t.Fatalf("duplicate identifier %q", value)
		}
		seen[value] = struct{}{}
		parts := strings.Split(value, "_")
		entropy := parts[len(parts)-1]
		if _, exists := entropySeen[entropy]; exists {
			t.Fatalf("duplicate entropy suffix %q", entropy)
		}
		entropySeen[entropy] = struct{}{}
	}
	if len(seen) != 32 {
		t.Fatalf("identifiers = %d, want 32", len(seen))
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("entropy unavailable") }

func TestPackageNewFallsBackOnEntropyFailure(t *testing.T) {
	previous := defaultGenerator
	defaultGenerator = NewGenerator(failingReader{}, time.Now)
	t.Cleanup(func() { defaultGenerator = previous })

	value := New("task")
	if value == "" || !strings.HasPrefix(value, "task_") {
		t.Fatalf("fallback identifier = %q", value)
	}
	if bytes.Contains([]byte(value), []byte("entropy")) {
		t.Fatalf("fallback identifier leaked error text: %q", value)
	}
}
