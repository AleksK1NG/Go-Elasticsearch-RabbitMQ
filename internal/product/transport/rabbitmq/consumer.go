package rabbitmq

import (
	"context"
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

func (c *consumer) ConsumeIndexDeliveries(ctx context.Context, deliveries <-chan amqp.Delivery) func() error {
	return func() error {
		c.log.Infof("starting consumer for queue deliveries: %s", c.cfg.ExchangeAndQueueBindings.IndexProductBinding.QueueName)

		for {
			select {
			case <-ctx.Done():
				c.log.Errorf("products consumer ctx done: %v", ctx.Err())
				return ctx.Err()

			case msg, ok := <-deliveries:
				if !ok {
					c.log.Errorf("NOT OK deliveries")
				}
				c.log.Infof("Consumer delivery: msg data: %s, headers: %+v", string(msg.Body), msg.Headers)
				if err := msg.Ack(true); err != nil {
					return err
				}
			}
		}

		c.log.Info("products consumer done")
		return nil
	}
}

func (c *consumer) ConsumeDeliveriesB(ctx context.Context, deliveries <-chan amqp.Delivery) func() error {
	return func() error {
		for delivery := range deliveries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			c.log.Infof("delivery: %s", string(delivery.Body))
			if err := delivery.Ack(true); err != nil {
				return err
			}
		}
		return nil
	}
}
