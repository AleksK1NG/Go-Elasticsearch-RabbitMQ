package app

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/pkg/rabbitmq"
	"github.com/avast/retry-go"
	"github.com/pkg/errors"
	"time"
)

const (
	prefetchCount        = 1
	prefetchSize         = 0
	qosGlobal            = true
	initRabbitMQAttempts = 5
	initRabbitMQDelay    = time.Duration(1500) * time.Millisecond
)

func (a *app) initRabbitMQ(ctx context.Context) error {
	retryOptions := []retry.Option{
		retry.Attempts(initRabbitMQAttempts),
		retry.Delay(initRabbitMQDelay),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			a.log.Errorf("retry connect rabbitmq err: %v", err)
		}),
	}

	return retry.Do(func() error {
		amqpConn, err := rabbitmq.NewRabbitMQConnection(a.cfg.RabbitMQ)
		if err != nil {
			return errors.Wrap(err, "rabbitmq.NewRabbitMQConnection")
		}
		a.amqpConn = amqpConn

		amqpChan, err := amqpConn.Channel()
		if err != nil {
			return errors.Wrap(err, "amqpConn.Channel")
		}
		a.amqpChan = amqpChan

		if err := a.amqpChan.Qos(prefetchCount, prefetchSize, qosGlobal); err != nil {
			a.log.Errorf("amqpChan.Qos err: %v", err)
			return errors.Wrap(err, "amqpChan.Qos")
		}

		a.log.Infof("rabbitmq connected: %+v", a.amqpConn)
		return nil
	}, retryOptions...)
}

func (a *app) closeRabbitConn() {
	if err := a.amqpChan.Close(); err != nil {
		a.log.Errorf("amqpChan.Close err: %v", err)
	}
	if err := a.amqpConn.Close(); err != nil {
		a.log.Errorf("amqpConn.Close err: %v", err)
	}
}

func (a *app) initRabbitMQPublisher(ctx context.Context) error {
	retryOptions := []retry.Option{
		retry.Attempts(initRabbitMQAttempts),
		retry.Delay(initRabbitMQDelay),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			a.log.Errorf("retry connect rabbitmq publisher err: %v", err)
		}),
	}

	return retry.Do(func() error {
		amqpPublisher, err := rabbitmq.NewPublisher(a.cfg.RabbitMQ, a.log)
		if err != nil {
			return errors.Wrap(err, "rabbitmq.NewPublisher")
		}
		a.amqpPublisher = amqpPublisher
		return nil

	}, retryOptions...)
}
