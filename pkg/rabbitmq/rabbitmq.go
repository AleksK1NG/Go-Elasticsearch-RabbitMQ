package rabbitmq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

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
