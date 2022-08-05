package usecase

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/utils"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "productUseCase.Index")
	defer span.Finish()
	span.LogFields(log.Object("product", product))
	return p.elasticRepository.Index(ctx, product)
}

func (p *productUseCase) Search(ctx context.Context, term string, pagination *utils.Pagination) (*domain.ProductSearchResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "productUseCase.Search")
	defer span.Finish()
	span.LogFields(log.String("term", term))
	return p.elasticRepository.Search(ctx, term, pagination)
}
