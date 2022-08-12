package usecase

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/rabbitmq"
	"github.com/AleksK1NG/go-elasticsearch/pkg/serializer"
	"github.com/AleksK1NG/go-elasticsearch/pkg/tracing"
	"github.com/AleksK1NG/go-elasticsearch/pkg/utils"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

type productUseCase struct {
	log               logger.Logger
	cfg               *config.Config
	elasticRepository domain.ElasticRepository
	amqpPublisher     rabbitmq.AmqpPublisher
}

func NewProductUseCase(
	log logger.Logger,
	cfg *config.Config,
	elasticRepository domain.ElasticRepository,
	amqpPublisher rabbitmq.AmqpPublisher,
) *productUseCase {
	return &productUseCase{log: log, cfg: cfg, elasticRepository: elasticRepository, amqpPublisher: amqpPublisher}
}

func (p *productUseCase) IndexAsync(ctx context.Context, product domain.Product) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "productUseCase.IndexAsync")
	defer span.Finish()
	span.LogFields(log.Object("product", product))

	dataBytes, err := serializer.Marshal(&product)
	if err != nil {
		return tracing.TraceWithErr(span, errors.Wrap(err, "serializer.Marshal"))
	}

	return p.amqpPublisher.Publish(
		ctx,
		p.cfg.ExchangeAndQueueBindings.IndexProductBinding.ExchangeName,
		p.cfg.ExchangeAndQueueBindings.IndexProductBinding.BindingKey,
		amqp.Publishing{
			Headers:   tracing.ExtractTextMapCarrierHeadersToAmqpTable(span.Context()),
			Timestamp: time.Now().UTC(),
			Body:      dataBytes,
		},
	)
}

func (p *productUseCase) Index(ctx context.Context, product domain.Product) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "productUseCase.Index")
	defer span.Finish()
	span.LogFields(log.Object("product", product))
	return p.elasticRepository.Index(ctx, product)
}

func (p *productUseCase) Search(ctx context.Context, term string, pagination *utils.Pagination) (*domain.ProductSearch, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "productUseCase.Search")
	defer span.Finish()
	span.LogFields(log.String("term", term))
	return p.elasticRepository.Search(ctx, term, pagination)
}
