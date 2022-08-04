package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
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
}

func NewConsumer(
	log logger.Logger,
	cfg *config.Config,
	amqpConn *amqp.Connection,
	amqpChan *amqp.Channel,
	productUseCase domain.ProductUseCase,
	esClient *elasticsearch.Client,
) *consumer {
	return &consumer{log: log, cfg: cfg, amqpConn: amqpConn, amqpChan: amqpChan, productUseCase: productUseCase, esClient: esClient}
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
					c.log.Errorf("NOT OK deliveries")
				}
				c.log.Infof("Consumer delivery: workerID: %d, msg data: %s, headers: %+v", workerID, string(msg.Body), msg.Headers)
				if err := c.bulkIndexProduct(ctx, msg); err != nil {
					return err
				}
				c.log.Infof("Consumer <<<ACK>>> delivery: workerID: %d, msg data: %s, headers: %+v", workerID, string(msg.Body), msg.Headers)
			}
		}

	}
}

func (c *consumer) indexProduct(ctx context.Context, msg amqp.Delivery) error {
	var product domain.Product
	if err := json.Unmarshal(msg.Body, &product); err != nil {
		c.log.Errorf("indexProduct json.Unmarshal <<<Reject>>> err: %v", err)
		return msg.Reject(true)
	}
	if err := c.productUseCase.Index(ctx, product); err != nil {
		c.log.Errorf("indexProduct productUseCase.Index <<<Reject>>> err: %v", err)
		return msg.Reject(true)
	}
	return msg.Ack(true)
}

func (c *consumer) bulkIndexProduct(ctx context.Context, msg amqp.Delivery) error {
	var product domain.Product
	if err := json.Unmarshal(msg.Body, &product); err != nil {
		c.log.Errorf("indexProduct json.Unmarshal <<<Reject>>> err: %v", err)
		return msg.Reject(true)
	}

	if err := c.bulkIndexer.Add(ctx, esutil.BulkIndexerItem{
		Index:      c.cfg.ElasticIndexes.ProductsIndex.Name,
		Action:     "create",
		DocumentID: product.ID,
		Body:       bytes.NewReader(msg.Body),
		OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, item2 esutil.BulkIndexerResponseItem) {
			c.log.Infof("bulk indexer onSuccess for index: %s", c.cfg.ElasticIndexes.ProductsIndex.Name)
		},
		OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, item2 esutil.BulkIndexerResponseItem, err error) {
			c.log.Errorf("bulk indexer OnFailure err: %v", err)
		},
	}); err != nil {
		c.log.Errorf("indexProduct bulkIndexer.Add <<<Reject>>> err: %v", err)
		return msg.Reject(true)
	}

	c.log.Infof("consumer <<<ACK>>> bulk indexer add product: %+v", product)
	return msg.Ack(true)
}

func (c *consumer) InitBulkIndexer() error {
	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		NumWorkers: 10,
		//FlushBytes:          0,
		FlushInterval: 5 * time.Second,
		Client:        c.esClient,
		OnError: func(ctx context.Context, err error) {
			c.log.Errorf("bulk indexer onError: %v", err)
		},
		OnFlushStart: func(ctx context.Context) context.Context {
			c.log.Infof("starting bulk indexer for index: %s", c.cfg.ElasticIndexes.ProductsIndex.Name)
			return ctx
		},
		OnFlushEnd: func(ctx context.Context) {
			c.log.Infof("finished bulk indexer for index: %s", c.cfg.ElasticIndexes.ProductsIndex.Name)
		},
		Index:   c.cfg.ElasticIndexes.ProductsIndex.Name,
		Human:   true,
		Pretty:  true,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return errors.Wrap(err, "esutil.NewBulkIndexer")
	}
	c.bulkIndexer = bulkIndexer
	c.log.Infof("consumer bulk indexer initialized for index: %s", c.cfg.ElasticIndexes.ProductsIndex.Name)
	return nil
}

func (c *consumer) Close(ctx context.Context) {
	if err := c.bulkIndexer.Close(ctx); err != nil {
		c.log.Errorf("bulk indexer Close: %v", err)
	}
}
