package rabbitmq

import (
	"context"
	"encoding/json"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

type consumer struct {
	log            logger.Logger
	cfg            *config.Config
	amqpConn       *amqp.Connection
	amqpChan       *amqp.Channel
	productUseCase domain.ProductUseCase
}

func NewConsumer(
	log logger.Logger,
	cfg *config.Config,
	amqpConn *amqp.Connection,
	amqpChan *amqp.Channel,
	productUseCase domain.ProductUseCase,
) *consumer {
	return &consumer{log: log, cfg: cfg, amqpConn: amqpConn, amqpChan: amqpChan, productUseCase: productUseCase}
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
				if err := c.indexProduct(ctx, msg); err != nil {
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
