package domain

import (
	"context"
	"time"
)

type Filter struct {
	Status     Status
	NodeID     string
	TemplateID string
	Limit      int
}

type Repository interface {
	Save(context.Context, *Execution) error
	Get(context.Context, string) (*Execution, error)
	Update(context.Context, *Execution) error
	List(context.Context, Filter) ([]*Execution, error)
	ListRunnable(context.Context, time.Time, int) ([]*Execution, error)
	ListActive(context.Context) ([]*Execution, error)
}
