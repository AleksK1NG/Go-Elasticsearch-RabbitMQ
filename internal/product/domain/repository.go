package domain

import "context"

type ElasticRepository interface {
	Index(ctx context.Context, product Product) error
}
