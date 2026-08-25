package id

import (
	"fmt"
	"strings"
	"time"
)

var defaultGenerator = &Generator{}

func New(prefix string) string {
	value, err := defaultGenerator.Next(prefix)
	if err != nil {
		return ""
	}
	return value
}

func fallback(prefix string) string {
	return fmt.Sprintf("%s_%d", clean(prefix), time.Now().UnixNano())
}

func clean(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "_")
	if value == "" {
		return "id"
	}
	return value
}
