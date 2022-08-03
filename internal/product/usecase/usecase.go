package usecase

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
)

type productUseCase struct {
	log               logger.Logger
	cfg               *config.Config
	elasticRepository domain.ElasticRepository
}

func NewProductUseCase(log logger.Logger, cfg *config.Config, elasticRepository domain.ElasticRepository) *productUseCase {
	return &productUseCase{log: log, cfg: cfg, elasticRepository: elasticRepository}
}

func (p *productUseCase) Index(ctx context.Context, product domain.Product) error {
	return p.elasticRepository.Index(ctx, product)
}

func (p *productUseCase) Search(ctx context.Context, term string) (any, error) {
	return p.elasticRepository.Search(ctx, term)
}
