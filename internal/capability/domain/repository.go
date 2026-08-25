package domain

import "context"

type Repository interface {
	Save(context.Context, *Capability) error
	ListByNode(context.Context, string) ([]*Capability, error)
	Find(context.Context, string, string) (*Capability, error)
	Delete(context.Context, string, string) error
}
