package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/pkg/errors"
	"time"
)

type esRepository struct {
	log      logger.Logger
	cfg      *config.Config
	esClient *elasticsearch.Client
}

func NewEsRepository(log logger.Logger, cfg *config.Config, esClient *elasticsearch.Client) *esRepository {
	return &esRepository{log: log, cfg: cfg, esClient: esClient}
}

func (e *esRepository) Index(ctx context.Context, product domain.Product) error {

	dataBytes, err := json.Marshal(&product)
	if err != nil {
		return errors.Wrap(err, "json.Marshal")
	}

	response, err := e.esClient.Index(
		e.cfg.ElasticIndexes.ProductsIndex.Name,
		bytes.NewReader(dataBytes),
		e.esClient.Index.WithPretty(),
		e.esClient.Index.WithHuman(),
		e.esClient.Index.WithTimeout(3*time.Second),
		e.esClient.Index.WithContext(ctx),
		e.esClient.Index.WithDocumentID(product.ID),
	)
	if err != nil {
		return errors.Wrap(err, "esClient.Index")
	}
	defer response.Body.Close()

	if response.IsError() {
		return errors.Wrap(errors.New(response.String()), "esClient.Index response error")
	}

	e.log.Infof("document indexed: %s", response.String())
	return nil
}
