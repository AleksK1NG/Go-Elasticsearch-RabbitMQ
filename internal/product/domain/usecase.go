package domain

import "context"

type ProductUseCase interface {
	Index(ctx context.Context, product Product) error
	Search(ctx context.Context, term string) (any, error)
}
