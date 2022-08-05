package elastic

import (
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"net/http"
	"os"
)

type BulkIndexerConfig struct {
	NumWorkers           int `mapstructure:"numWorkers" validate:"required"`
	FlushBytes           int `mapstructure:"flushBytes" validate:"required"`
	FlushIntervalSeconds int `mapstructure:"flushIntervalSeconds" validate:"required"`
	TimeoutMilliseconds  int `mapstructure:"timeoutMilliseconds" validate:"required"`
}

type Config struct {
	Addresses []string `mapstructure:"addresses" validate:"required"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`

	APIKey        string      `mapstructure:"apiKey"`
	Header        http.Header // Global HTTP request header.
	EnableLogging bool        `mapstructure:"enableLogging"`
}

func NewElasticSearchClient(cfg Config) (*elasticsearch.Client, error) {

	config := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		APIKey:    cfg.APIKey,
		Header:    cfg.Header,
	}

	if cfg.EnableLogging {
		config.Logger = &elastictransport.ColorLogger{Output: os.Stdout, EnableRequestBody: true, EnableResponseBody: true}
	}

	client, err := elasticsearch.NewClient(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
