package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// New creates a sortable-enough identifier without a global counter.
func New(prefix string) string {
	var random [6]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Sprintf("%s_%d", clean(prefix), time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%d_%s", clean(prefix), time.Now().UTC().UnixMilli(), hex.EncodeToString(random[:]))
}

func clean(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "_")
	if value == "" {
		return "id"
	}
	return value
}
