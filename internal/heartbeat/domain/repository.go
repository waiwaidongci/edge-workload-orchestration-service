package domain

import (
	"context"
	"time"
)

type Repository interface {
	Append(context.Context, Record) error
	Recent(context.Context, string, time.Time, int) ([]Record, error)
	AppendStatusChange(context.Context, StatusChange) error
	StatusChanges(context.Context, string, int) ([]StatusChange, error)
}

type Presence interface {
	Touch(context.Context, string, time.Duration) error
	Online(context.Context, string) (bool, error)
	Remove(context.Context, string) error
}
