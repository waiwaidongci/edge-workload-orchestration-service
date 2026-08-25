package domain

import "context"

type Filter struct {
	Region string
	Status Status
	Labels map[string]string
}

type Repository interface {
	Save(context.Context, *Node) error
	Get(context.Context, string) (*Node, error)
	List(context.Context, Filter) ([]*Node, error)
	Update(context.Context, *Node) error
}
