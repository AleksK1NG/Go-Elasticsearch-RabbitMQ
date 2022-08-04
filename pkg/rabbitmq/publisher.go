package rabbitmq

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

type AmqpPublisher interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Publish(ctx context.Context, exchange, key string, msg amqp.Publishing) error
	Close()
}

type publisher struct {
	log      logger.Logger
	amqpConn *amqp.Connection
	amqpChan *amqp.Channel
}

func NewPublisher(cfg Config, log logger.Logger) (*publisher, error) {
	conn, err := NewRabbitMQConnection(cfg)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &publisher{
		log:      log,
		amqpConn: conn,
		amqpChan: channel,
	}, nil
}

func (p *publisher) Close() {
	if err := p.amqpChan.Close(); err != nil {
		p.log.Errorf("publisher mqpChan.Close err: %v", err)
	}
	if err := p.amqpConn.Close(); err != nil {
		p.log.Errorf("publisher amqpConn.Close err: %v", err)
	}
}

func (p *publisher) Publish(ctx context.Context, exchange, key string, msg amqp.Publishing) error {
	if err := p.amqpChan.PublishWithContext(
		ctx,
		exchange,
		key,
		false,
		false,
		msg,
	); err != nil {
		p.log.Errorf("publisher Publish err: %v", err)
		return err
	}
	return nil
}

func (p *publisher) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	if err := p.amqpChan.PublishWithContext(
		ctx,
		exchange,
		key,
		mandatory,
		immediate,
		msg,
	); err != nil {
		p.log.Errorf("publisher PublishWithContext err: %v", err)
		return err
	}
	return nil
}
