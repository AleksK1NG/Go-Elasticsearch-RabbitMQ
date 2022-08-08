package app

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/pkg/elastic"
	"github.com/avast/retry-go"
	"time"
)

func (a *app) initElasticsearchClient(ctx context.Context) error {
	retryOptions := []retry.Option{
		retry.Attempts(5),
		retry.Delay(time.Duration(1500) * time.Millisecond),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			a.log.Errorf("retry connect elasticsearch err: %v", err)
		}),
	}

	return retry.Do(func() error {
		elasticSearchClient, err := elastic.NewElasticSearchClient(a.cfg.ElasticSearch)
		if err != nil {
			return err
		}
		a.elasticClient = elasticSearchClient
		return nil
	}, retryOptions...)
}
