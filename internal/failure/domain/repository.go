package domain

import "context"

type Repository interface {
	Save(context.Context, *DeadLetter) error
	Get(context.Context, string) (*DeadLetter, error)
	List(context.Context, bool, int) ([]*DeadLetter, error)
	Update(context.Context, *DeadLetter) error
}
