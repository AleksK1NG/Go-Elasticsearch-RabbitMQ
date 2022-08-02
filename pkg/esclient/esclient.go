package esclient

import (
	"context"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/pkg/errors"
)

func Info(ctx context.Context, esClient *elasticsearch.Client) (*esapi.Response, error) {
	response, err := esClient.Info(esClient.Info.WithContext(ctx), esClient.Info.WithHuman())
	if err != nil {
		return nil, err
	}
	if response.IsError() {
		return nil, errors.New(response.String())
	}

	return response, nil
}
