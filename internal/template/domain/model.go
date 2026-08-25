package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	nodedomain "github.com/example/edge-task-orchestrator/internal/node/domain"
)

var ErrNotFound = errors.New("task template not found")
var ErrVersionNotFound = errors.New("task template version not found")

type RetryPolicy struct {
	MaxAttempts        int `json:"max_attempts"`
	BaseBackoffSeconds int `json:"base_backoff_seconds"`
}

type Version struct {
	Number               int                  `json:"number"`
	Image                string               `json:"image"`
	Command              []string             `json:"command"`
	Environment          map[string]string    `json:"environment"`
	Resources            nodedomain.Resources `json:"resources"`
	RequiredCapabilities []string             `json:"required_capabilities"`
	TimeoutSeconds       int                  `json:"timeout_seconds"`
	Retry                RetryPolicy          `json:"retry"`
	CreatedAt            time.Time            `json:"created_at"`
}

type Template struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Versions      []Version `json:"versions"`
	LatestVersion int       `json:"latest_version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func New(identifier, name, description string, now time.Time) (*Template, error) {
	template := &Template{ID: strings.TrimSpace(identifier), Name: strings.TrimSpace(name), Description: strings.TrimSpace(description), CreatedAt: now, UpdatedAt: now}
	if template.ID == "" || template.Name == "" {
		return nil, fmt.Errorf("template id and name are required")
	}
	return template, nil
}

func (t *Template) AddVersion(version Version, now time.Time) (Version, error) {
	if strings.TrimSpace(version.Image) == "" {
		return Version{}, fmt.Errorf("image is required")
	}
	if version.TimeoutSeconds <= 0 {
		return Version{}, fmt.Errorf("timeout_seconds must be positive")
	}
	if version.Retry.MaxAttempts <= 0 {
		version.Retry.MaxAttempts = 1
	}
	if version.Retry.BaseBackoffSeconds <= 0 {
		version.Retry.BaseBackoffSeconds = 1
	}
	if err := version.Resources.Validate(); err != nil {
		return Version{}, fmt.Errorf("resources: %w", err)
	}
	version.Number = t.LatestVersion + 1
	version.CreatedAt = now
	version.RequiredCapabilities = unique(version.RequiredCapabilities)
	version = copyVersion(version)
	t.Versions = append(t.Versions, version)
	t.LatestVersion = version.Number
	t.UpdatedAt = now
	return version, nil
}

func (t *Template) Version(number int) (Version, error) {
	if number == 0 {
		number = t.LatestVersion
	}
	for _, version := range t.Versions {
		if version.Number == number {
			return copyVersion(version), nil
		}
	}
	return Version{}, ErrVersionNotFound
}

func unique(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
func copyVersion(source Version) Version {
	out := source
	if source.Command != nil {
		out.Command = append([]string(nil), source.Command...)
	}
	if source.RequiredCapabilities != nil {
		out.RequiredCapabilities = append([]string(nil), source.RequiredCapabilities...)
	}
	if source.Environment != nil {
		out.Environment = make(map[string]string, len(source.Environment))
		for k, v := range source.Environment {
			out.Environment[k] = v
		}
	}
	return out
}

// CopyVersion returns a deep copy of v, detaching all slice and map fields
// from any aliasing the caller may hold. It is the package's public deep-copy
// primitive for the Version type, used by callers that need ownership of the
// copied data (e.g. the infrastructure layer when cloning a Template).
func CopyVersion(v Version) Version { return copyVersion(v) }
