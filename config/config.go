package config

import (
	"flag"
	"fmt"
	"github.com/AleksK1NG/go-elasticsearch/pkg/constants"
	"github.com/AleksK1NG/go-elasticsearch/pkg/elastic"
	"github.com/AleksK1NG/go-elasticsearch/pkg/esclient"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/probes"
	"github.com/AleksK1NG/go-elasticsearch/pkg/rabbitmq"
	"github.com/AleksK1NG/go-elasticsearch/pkg/tracing"
	"github.com/pkg/errors"
	"os"

	"github.com/spf13/viper"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "", "Search microservice config path")
}

type Config struct {
	ServiceName              string                    `mapstructure:"serviceName"`
	Logger                   logger.LogConfig          `mapstructure:"logger"`
	GRPC                     GRPC                      `mapstructure:"grpc"`
	Timeouts                 Timeouts                  `mapstructure:"timeouts" validate:"required"`
	Jaeger                   *tracing.Config           `mapstructure:"jaeger"`
	ElasticIndexes           ElasticIndexes            `mapstructure:"elasticIndexes" validate:"required"`
	Http                     Http                      `mapstructure:"http"`
	Probes                   probes.Config             `mapstructure:"probes"`
	ElasticSearch            elastic.Config            `mapstructure:"elasticSearch" validate:"required"`
	RabbitMQ                 rabbitmq.Config           `mapstructure:"rabbitmq" validate:"required"`
	ExchangeAndQueueBindings ExchangeAndQueueBindings  `mapstructure:"exchangeAndQueueBindings" validate:"required"`
	BulkIndexerConfig        elastic.BulkIndexerConfig `mapstructure:"bulkIndexer" validate:"required"`
}

type ExchangeAndQueueBindings struct {
	IndexProductBinding rabbitmq.ExchangeAndQueueBinding `mapstructure:"indexProductBinding" validate:"required"`
}

type GRPC struct {
	Port        string `mapstructure:"port"`
	Development bool   `mapstructure:"development"`
}

type Timeouts struct {
	PostgresInitMilliseconds int  `mapstructure:"postgresInitMilliseconds" validate:"required"`
	PostgresInitRetryCount   uint `mapstructure:"postgresInitRetryCount" validate:"required"`
}

type Projections struct {
	MongoGroup                  string `mapstructure:"mongoGroup" validate:"required"`
	MongoSubscriptionPoolSize   int    `mapstructure:"mongoSubscriptionPoolSize" validate:"required,gte=0"`
	ElasticGroup                string `mapstructure:"elasticGroup" validate:"required"`
	ElasticSubscriptionPoolSize int    `mapstructure:"elasticSubscriptionPoolSize" validate:"required,gte=0"`
}

type Http struct {
	Port                string   `mapstructure:"port" validate:"required"`
	Development         bool     `mapstructure:"development"`
	BasePath            string   `mapstructure:"basePath" validate:"required"`
	ProductsPath        string   `mapstructure:"productsPath" validate:"required"`
	DebugErrorsResponse bool     `mapstructure:"debugErrorsResponse"`
	IgnoreLogUrls       []string `mapstructure:"ignoreLogUrls"`
}

type ElasticIndexes struct {
	ProductsIndex esclient.ElasticIndex `mapstructure:"products" validate:"required"`
}

func InitConfig() (*Config, error) {
	if configPath == "" {
		configPathFromEnv := os.Getenv(constants.ConfigPath)
		if configPathFromEnv != "" {
			configPath = configPathFromEnv
		} else {
			getwd, err := os.Getwd()
			if err != nil {
				return nil, errors.Wrap(err, "os.Getwd")
			}
			configPath = fmt.Sprintf("%s/config/config.yaml", getwd)
		}
	}

	cfg := &Config{}

	viper.SetConfigType(constants.Yaml)
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "viper.ReadInConfig")
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, errors.Wrap(err, "viper.Unmarshal")
	}

	grpcPort := os.Getenv(constants.GrpcPort)
	if grpcPort != "" {
		cfg.GRPC.Port = grpcPort
	}

	jaegerAddr := os.Getenv(constants.JaegerHostPort)
	if jaegerAddr != "" {
		cfg.Jaeger.HostPort = jaegerAddr
	}

	elasticUrl := os.Getenv(constants.ElasticUrl)
	if elasticUrl != "" {
		cfg.ElasticSearch.Addresses = []string{elasticUrl}
	}

	rabbitURI := os.Getenv(constants.RabbitMQ_URI)
	if elasticUrl != "" {
		cfg.RabbitMQ.URI = rabbitURI
	}

	return cfg, nil
}
