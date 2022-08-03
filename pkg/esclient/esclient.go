package esclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/pkg/errors"
	"time"
)

type ElasticIndex struct {
	Path  string `mapstructure:"path" validate:"required"`
	Name  string `mapstructure:"name" validate:"required"`
	Alias string `mapstructure:"alias" validate:"required"`
}

func (e *ElasticIndex) String() string {
	return fmt.Sprintf("Name: %s, Alias: %s, Path: %s", e.Name, e.Alias, e.Path)
}

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

func CreateIndex(ctx context.Context, esClient *elasticsearch.Client, name string, data []byte) (*esapi.Response, error) {
	response, err := esClient.Indices.Create(
		name,
		esClient.Indices.Create.WithContext(ctx),
		esClient.Indices.Create.WithBody(bytes.NewReader(data)),
		esClient.Indices.Create.WithPretty(),
		esClient.Indices.Create.WithHuman(),
		esClient.Indices.Create.WithTimeout(3*time.Second),
	)
	if err != nil {
		return nil, err
	}

	if response.IsError() {
		return nil, errors.New(response.String())
	}

	return response, nil
}

func CreateAlias(ctx context.Context, esClient *elasticsearch.Client, indexes []string, name string, data []byte) (*esapi.Response, error) {
	response, err := esClient.Indices.PutAlias(
		indexes,
		name,
		esClient.Indices.PutAlias.WithBody(bytes.NewReader(data)),
		esClient.Indices.PutAlias.WithContext(ctx),
		esClient.Indices.PutAlias.WithHuman(),
		esClient.Indices.PutAlias.WithPretty(),
		esClient.Indices.PutAlias.WithTimeout(3*time.Second),
	)
	if err != nil {
		return nil, err
	}

	if response.IsError() {
		return nil, errors.New(response.String())
	}

	return response, nil
}

func Exists(ctx context.Context, esClient *elasticsearch.Client, indexes []string) (*esapi.Response, error) {
	response, err := esClient.Indices.Exists(
		indexes,
		esClient.Indices.Exists.WithContext(ctx),
		esClient.Indices.Exists.WithHuman(),
		esClient.Indices.Exists.WithPretty(),
	)
	if err != nil {
		return nil, err
	}

	if response.IsError() && response.StatusCode != 404 {
		return nil, errors.New(response.String())
	}

	return response, nil
}

func Search(ctx context.Context, esClient *elasticsearch.Client, index, term string, fields []string) (*esapi.Response, error) {
	//query := make(map[string]any, 5)
	//multiMatch := map[string]any{"multi_match": map[string]any{
	//	"query":  term,
	//	"fields": fields,
	//}}
	//query["bool"] = map[string]any{"must": []map[string]any{multiMatch}}
	//
	//multiMatchBytes, err := json.Marshal(&multiMatch)
	//if err != nil {
	//	return nil, err
	//}
	//
	//log.Printf("json search: %s", string(multiMatchBytes))
	//log.Printf("json search: %s", string(multiMatchBytes))

	query := MultiMatchQuery{
		Query: Query{
			Bool: Bool{
				Must: []any{MultiMatch{
					Query:  term,
					Fields: fields,
				}},
			},
		},
	}

	dataBytes, err := json.Marshal(&query)
	if err != nil {
		return nil, err
	}

	response, err := esClient.Search(
		esClient.Search.WithContext(ctx),
		esClient.Search.WithIndex(index),
		esClient.Search.WithBody(bytes.NewReader(dataBytes)),
		esClient.Search.WithPretty(),
		esClient.Search.WithHuman(),
		esClient.Search.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, err
	}

	if response.IsError() {
		return nil, errors.Wrap(errors.New(response.String()), "esClient.Search error")
	}

	return response, nil
}
