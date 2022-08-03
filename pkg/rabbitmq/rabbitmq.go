package rabbitmq

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"
	"log"
)

type ExchangeConfig struct {
	Name string `mapstructure:"name" validate:"required"`
	Kind string `mapstructure:"kind" validate:"required"`
}

type QueueConfig struct {
	Name string `mapstructure:"name" validate:"required"`
}

type ExchangeAndQueueBinding struct {
	ExchangeName string `mapstructure:"exchangeName" validate:"required"`
	ExchangeKind string `mapstructure:"exchangeKind" validate:"required"`
	QueueName    string `mapstructure:"queueName" validate:"required"`
	BindingKey   string `mapstructure:"bindingKey" validate:"required"`
}

type Config struct {
	URI string `mapstructure:"uri" validate:"required"`
}

func NewRabbitMQ(cfg Config) (*amqp.Connection, error) {
	conn, err := amqp.Dial(cfg.URI)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func DeclareBinding(ctx context.Context, channel *amqp.Channel, exchangeAndQueueBinding ExchangeAndQueueBinding) (amqp.Queue, error) {
	if err := DeclareExchange(ctx, channel, exchangeAndQueueBinding.ExchangeName, exchangeAndQueueBinding.ExchangeKind); err != nil {
		return amqp.Queue{}, err
	}

	queue, err := DeclareQueue(ctx, channel, exchangeAndQueueBinding.QueueName)
	if err != nil {
		return amqp.Queue{}, err
	}

	if err := BindQueue(ctx, channel, queue.Name, exchangeAndQueueBinding.BindingKey, exchangeAndQueueBinding.ExchangeName); err != nil {
		return amqp.Queue{}, err
	}

	return queue, nil
}

func DeclareExchange(ctx context.Context, channel *amqp.Channel, name, kind string) error {
	return channel.ExchangeDeclare(
		name,
		kind,
		true,
		false,
		false,
		false,
		nil,
	)
}

func DeclareQueue(ctx context.Context, channel *amqp.Channel, name string) (amqp.Queue, error) {
	return channel.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		nil,
	)
}

func BindQueue(ctx context.Context, channel *amqp.Channel, queue, key, exchange string) error {
	return channel.QueueBind(queue, key, exchange, false, nil)
}

func ConsumeQueue(ctx context.Context, channel *amqp.Channel, concurrency int) error {
	deliveries, err := channel.Consume(
		"queue",
		"consumer",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	for i := 0; i <= concurrency; i++ {
		eg.Go(processDeliveries(ctx, deliveries))
	}

	return eg.Wait()
}

func processDeliveries(ctx context.Context, deliveries <-chan amqp.Delivery) func() error {
	return func() error {
		for delivery := range deliveries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			log.Printf("delivery: %s", string(delivery.Body))
			if err := delivery.Ack(true); err != nil {
				return err
			}
		}
		return nil
	}
}
