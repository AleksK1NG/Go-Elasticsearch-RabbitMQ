package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/esclient"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/misstype_manager"
	"github.com/AleksK1NG/go-elasticsearch/pkg/utils"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"time"
)

type esRepository struct {
	log             logger.Logger
	cfg             *config.Config
	esClient        *elasticsearch.Client
	missTypeManager misstype_manager.MissTypeManager
}

func NewEsRepository(
	log logger.Logger,
	cfg *config.Config,
	esClient *elasticsearch.Client,
	missTypeManager misstype_manager.MissTypeManager,
) *esRepository {
	return &esRepository{log: log, cfg: cfg, esClient: esClient, missTypeManager: missTypeManager}
}

func (e *esRepository) Index(ctx context.Context, product domain.Product) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "esRepository.Index")
	defer span.Finish()
	span.LogFields(log.Object("product", product))

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

func (e *esRepository) Search(ctx context.Context, term string, pagination *utils.Pagination) (*domain.ProductSearchResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "esRepository.Search")
	defer span.Finish()
	span.LogFields(log.String("term", term))

	shouldQuery := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"should": []map[string]any{
					{
						"multi_match": map[string]any{
							"query":  term,
							"fields": []string{"title", "description"},
						}},
					{
						"multi_match": map[string]any{
							"query":  e.missTypeManager.GetMissTypedWord(term),
							"fields": []string{"title", "description"},
						},
					},
				},
			},
		},
	}

	dataBytes, err := json.Marshal(&shouldQuery)
	if err != nil {
		return nil, err
	}

	e.log.Infof("JSON QUERY : %+v", shouldQuery)
	e.log.Infof("JSON BODY: %s", string(dataBytes))

	response, err := e.esClient.Search(
		e.esClient.Search.WithContext(ctx),
		e.esClient.Search.WithIndex(e.cfg.ElasticIndexes.ProductsIndex.Name),
		e.esClient.Search.WithBody(bytes.NewReader(dataBytes)),
		e.esClient.Search.WithPretty(),
		e.esClient.Search.WithHuman(),
		e.esClient.Search.WithTimeout(5*time.Second),
		e.esClient.Search.WithSize(pagination.GetSize()),
		e.esClient.Search.WithFrom(pagination.GetOffset()),
	)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.IsError() {
		return nil, errors.Wrap(errors.New(response.String()), "esClient.Search error")
	}

	e.log.Infof("repository search result: %s", response.String())

	hits := esclient.EsHits[*domain.Product]{}
	if err := json.NewDecoder(response.Body).Decode(&hits); err != nil {
		return nil, err
	}

	responseList := make([]*domain.Product, len(hits.Hits.Hits))
	for i, source := range hits.Hits.Hits {
		responseList[i] = source.Source
	}

	e.log.Infof("repository search result responseList: %+v", responseList)
	return &domain.ProductSearchResponse{
		List:               responseList,
		PaginationResponse: utils.NewPaginationResponse(hits.Hits.Total.Value, pagination),
	}, nil
}
