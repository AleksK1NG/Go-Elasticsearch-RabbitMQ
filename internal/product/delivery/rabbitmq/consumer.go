package rabbitmq

import (
	"bytes"
	"context"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/metrics"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/serializer"
	"github.com/AleksK1NG/go-elasticsearch/pkg/tracing"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

type consumer struct {
	log            logger.Logger
	cfg            *config.Config
	amqpConn       *amqp.Connection
	amqpChan       *amqp.Channel
	productUseCase domain.ProductUseCase
	bulkIndexer    esutil.BulkIndexer
	esClient       *elasticsearch.Client
	metrics        *metrics.SearchMicroserviceMetrics
}

func NewConsumer(
	log logger.Logger,
	cfg *config.Config,
	amqpConn *amqp.Connection,
	amqpChan *amqp.Channel,
	productUseCase domain.ProductUseCase,
	esClient *elasticsearch.Client,
	metrics *metrics.SearchMicroserviceMetrics,
) *consumer {
	return &consumer{log: log, cfg: cfg, amqpConn: amqpConn, amqpChan: amqpChan, productUseCase: productUseCase, esClient: esClient, metrics: metrics}
}

func (c *consumer) ConsumeIndexDeliveries(ctx context.Context, deliveries <-chan amqp.Delivery, workerID int) func() error {
	return func() error {
		c.log.Infof("starting consumer workerID: %d, for queue deliveries: %s", workerID, c.cfg.ExchangeAndQueueBindings.IndexProductBinding.QueueName)

		for {
			select {
			case <-ctx.Done():
				c.log.Errorf("products consumer ctx done: %v", ctx.Err())
				return ctx.Err()

			case msg, ok := <-deliveries:
				if !ok {
					c.log.Errorf("NOT OK deliveries channel closed for queue: %s", c.cfg.ExchangeAndQueueBindings.IndexProductBinding.QueueName)
					return errors.New("deliveries channel closed")
				}

				c.log.Infof("Consumer delivery: workerID: %d, msg data: %s, headers: %+v", workerID, string(msg.Body), msg.Headers)

				if err := c.bulkIndexProduct(ctx, msg); err != nil {
					c.log.Errorf("bulkIndexProduct err: %v", err)
					continue
				}

				c.log.Infof("Consumer <<<ACK>>> delivery: workerID: %d, msg data: %s, headers: %+v", workerID, string(msg.Body), msg.Headers)
				c.metrics.RabbitMQSuccessBatchInsertMessages.Inc()
			}
		}

	}
}

func (c *consumer) indexProduct(ctx context.Context, msg amqp.Delivery) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "consumer.indexProduct")
	defer span.Finish()

	var product domain.Product
	if err := serializer.Unmarshal(msg.Body, &product); err != nil {
		c.log.Errorf("indexProduct serializer.Unmarshal <<<Reject>>> err: %v", tracing.TraceWithErr(span, err))
		return msg.Reject(true)
	}
	if err := c.productUseCase.Index(ctx, product); err != nil {
		c.log.Errorf("indexProduct productUseCase.Index <<<Reject>>> err: %v", tracing.TraceWithErr(span, err))
		return msg.Reject(true)
	}
	return msg.Ack(true)
}

func (c *consumer) bulkIndexProduct(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := tracing.StartRabbitConsumerTracerSpan(ctx, msg.Headers, "consumer.bulkIndexProduct")
	defer span.Finish()

	var product domain.Product
	if err := serializer.Unmarshal(msg.Body, &product); err != nil {
		c.log.Errorf("indexProduct serializer.Unmarshal <<<Reject>>> err: %v", tracing.TraceWithErr(span, err))
		return msg.Reject(true)
	}

	if err := c.bulkIndexer.Add(ctx, esutil.BulkIndexerItem{
		Index:      c.cfg.ElasticIndexes.ProductsIndex.Name,
		Action:     "index",
		DocumentID: product.ID,
		Body:       bytes.NewReader(msg.Body),
		OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, item2 esutil.BulkIndexerResponseItem) {
			c.log.Debugf("bulk indexer onSuccess for index alias: %s", c.cfg.ElasticIndexes.ProductsIndex.Alias)
		},
		OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, item2 esutil.BulkIndexerResponseItem, err error) {
			if err != nil {
				c.log.Errorf("bulk indexer OnFailure err: %v", err)
			}
		},
	}); err != nil {
		c.log.Errorf("indexProduct bulkIndexer.Add <<<Reject>>> err: %v", tracing.TraceWithErr(span, err))
		return msg.Reject(true)
	}

	c.log.Infof("consumer <<<ACK>>> bulk indexer add product: %+v", product)
	return msg.Ack(true)
}

func (c *consumer) InitBulkIndexer() error {
	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		NumWorkers:    c.cfg.BulkIndexerConfig.NumWorkers,
		FlushBytes:    c.cfg.BulkIndexerConfig.FlushBytes,
		FlushInterval: time.Duration(c.cfg.BulkIndexerConfig.FlushIntervalSeconds) * time.Second,
		Client:        c.esClient,
		OnError: func(ctx context.Context, err error) {
			c.log.Errorf("bulk indexer onError: %v", err)
		},
		OnFlushStart: func(ctx context.Context) context.Context {
			c.log.Infof("starting bulk indexer for index alias: %s", c.cfg.ElasticIndexes.ProductsIndex.Alias)
			return ctx
		},
		OnFlushEnd: func(ctx context.Context) {
			c.log.Infof("finished bulk indexer for index alias: %s", c.cfg.ElasticIndexes.ProductsIndex.Alias)
		},
		Index:   c.cfg.ElasticIndexes.ProductsIndex.Alias,
		Human:   true,
		Pretty:  true,
		Timeout: time.Duration(c.cfg.BulkIndexerConfig.TimeoutMilliseconds) * time.Second,
	})
	if err != nil {
		return errors.Wrap(err, "esutil.NewBulkIndexer")
	}

	c.bulkIndexer = bulkIndexer
	c.log.Infof("consumer bulk indexer initialized for index alias: %s", c.cfg.ElasticIndexes.ProductsIndex.Alias)
	return nil
}

func (c *consumer) Close(ctx context.Context) {
	if err := c.bulkIndexer.Close(ctx); err != nil {
		c.log.Errorf("bulk indexer Close: %v", err)
	}
}
