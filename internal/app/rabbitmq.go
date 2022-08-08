package app

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/pkg/rabbitmq"
	"github.com/avast/retry-go"
	"time"
)

func (a *app) initRabbitMQ(ctx context.Context) error {
	retryOptions := []retry.Option{
		retry.Attempts(5),
		retry.Delay(time.Duration(1500) * time.Millisecond),
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
			return err
		}
		//defer amqpConn.Close()
		a.amqpConn = amqpConn

		amqpChan, err := amqpConn.Channel()
		if err != nil {
			return err
		}
		//defer amqpChan.Close()
		a.amqpChan = amqpChan

		if err := a.amqpChan.Qos(1, 0, true); err != nil {
			a.log.Errorf("amqpChan.Qos err: %v", err)
			return err
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
		retry.Attempts(5),
		retry.Delay(time.Duration(1500) * time.Millisecond),
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
			return err
		}
		a.amqpPublisher = amqpPublisher
		return nil

	}, retryOptions...)
}
