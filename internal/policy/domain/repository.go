package domain

import "context"

type Repository interface {
	Save(context.Context, *Policy) error
	Get(context.Context, string) (*Policy, error)
	List(context.Context) ([]*Policy, error)
	Update(context.Context, *Policy) error
}
