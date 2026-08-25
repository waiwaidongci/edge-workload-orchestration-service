package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"
)

type Generator struct {
	mu      sync.Mutex
	entropy io.Reader
	now     func() time.Time
	seen    map[string]uint64
	scratch [6]byte
}

func NewGenerator(entropy io.Reader, now func() time.Time) *Generator {
	return &Generator{entropy: entropy, now: now, seen: make(map[string]uint64)}
}

func (g *Generator) Next(prefix string) (string, error) {
	g.initialize()
	key := clean(prefix)
	g.seen[key]++
	sequence := g.seen[key]
	timestamp := g.timestamp()

	random, err := g.readEntropy()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%d_%d_%s", key, timestamp, sequence, hex.EncodeToString(random[:])), nil
}

func (g *Generator) initialize() {
	if g.entropy == nil {
		g.entropy = rand.Reader
	}
	if g.now == nil {
		g.now = time.Now
	}
}

func (g *Generator) timestamp() int64 {
	observed := g.now().UTC()
	return observed.UnixMilli()
}

func (g *Generator) readEntropy() ([6]byte, error) {
	if _, err := g.entropy.Read(g.scratch[:]); err != nil {
		return [6]byte{}, fmt.Errorf("read identifier entropy: %w", err)
	}
	return g.scratch, nil
}
