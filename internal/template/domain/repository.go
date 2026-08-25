package domain

import "context"

type Repository interface {
	Save(context.Context, *Template) error
	Get(context.Context, string) (*Template, error)
	List(context.Context) ([]*Template, error)
	Update(context.Context, *Template) error
}
