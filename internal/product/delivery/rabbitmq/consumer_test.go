package rabbitmq

import (
	"context"
	"encoding/json"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/rabbitmq"
	"github.com/brianvoe/gofakeit/v6"
	amqp "github.com/rabbitmq/amqp091-go"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestConsumer_ConsumeIndexDeliveries(t *testing.T) {
	t.Parallel()

	appLogger := logger.NewAppLogger(logger.LogConfig{
		LogLevel: "debug",
		DevMode:  false,
		Encoder:  "console",
	})
	appLogger.InitLogger()
	appLogger.Named("Publisher")

	amqpPublisher, err := rabbitmq.NewPublisher(rabbitmq.Config{URI: "amqp://guest:guest@localhost:5672/"}, appLogger)
	if err != nil {
		require.NoError(t, err)

	}
	defer amqpPublisher.Close()

	for i := 0; i < 1500; i++ {
		//time.Sleep(10 * time.Millisecond)

		product := domain.Product{
			ID:           uuid.NewV4().String(),
			Title:        gofakeit.Fruit(),
			Description:  gofakeit.AdjectiveDescriptive(),
			ImageURL:     gofakeit.URL(),
			CountInStock: gofakeit.Int64(),
			Shop:         gofakeit.Company(),
			CreatedAt:    time.Now().UTC(),
		}

		dataBytes, err := json.Marshal(&product)
		if err != nil {
			require.NoError(t, err)
		}

		if err := amqpPublisher.Publish(
			context.Background(),
			"products",
			"products-index",
			amqp.Publishing{
				Headers:   map[string]interface{}{"trace-id": uuid.NewV4().String()},
				Timestamp: time.Now().UTC(),
				Body:      dataBytes,
			},
		); err != nil {
			require.NoError(t, err)
		}
		appLogger.Infof("message published: %+v", product)
	}

}
