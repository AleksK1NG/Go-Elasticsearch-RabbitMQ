package domain

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/pkg/utils"
)

type ElasticRepository interface {
	Index(ctx context.Context, product Product) error
	Search(ctx context.Context, term string, pagination *utils.Pagination) (*ProductSearch, error)
}
