package repository

import (
	"bufio"
	"bytes"
	"context"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/esclient"
	"github.com/AleksK1NG/go-elasticsearch/pkg/keyboard_manager"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/serializer"
	"github.com/AleksK1NG/go-elasticsearch/pkg/tracing"
	"github.com/AleksK1NG/go-elasticsearch/pkg/utils"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"time"
)

const (
	indexTimeout  = 5 * time.Second
	searchTimeout = 5 * time.Second
)

var (
	searchFields = []string{"title", "description"}
)

type esRepository struct {
	log                   logger.Logger
	cfg                   *config.Config
	esClient              *elasticsearch.Client
	keyboardLayoutManager keyboard_manager.KeyboardLayoutManager
}

func NewEsRepository(
	log logger.Logger,
	cfg *config.Config,
	esClient *elasticsearch.Client,
	missTypeManager keyboard_manager.KeyboardLayoutManager,
) *esRepository {
	return &esRepository{log: log, cfg: cfg, esClient: esClient, keyboardLayoutManager: missTypeManager}
}

func (e *esRepository) Index(ctx context.Context, product domain.Product) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "esRepository.Index")
	defer span.Finish()
	span.LogFields(log.Object("product", product))

	dataBytes, err := serializer.Marshal(&product)
	if err != nil {
		return tracing.TraceWithErr(span, errors.Wrap(err, "serializer.Marshal"))
	}

	response, err := e.esClient.Index(
		e.cfg.ElasticIndexes.ProductsIndex.Alias,
		bytes.NewReader(dataBytes),
		e.esClient.Index.WithPretty(),
		e.esClient.Index.WithHuman(),
		e.esClient.Index.WithTimeout(indexTimeout),
		e.esClient.Index.WithContext(ctx),
		e.esClient.Index.WithDocumentID(product.ID),
	)
	if err != nil {
		return tracing.TraceWithErr(span, errors.Wrap(err, "esClient.Index"))
	}
	defer response.Body.Close()

	if response.IsError() {
		return tracing.TraceWithErr(span, errors.Wrap(errors.New(response.String()), "esClient.Index response error"))
	}

	e.log.Infof("document indexed: %s", response.String())
	return nil
}

func (e *esRepository) Search(ctx context.Context, term string, pagination *utils.Pagination) (*domain.ProductSearch, error) {
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
							"fields": searchFields,
						},
					},
					{
						"multi_match": map[string]any{
							"query":  e.keyboardLayoutManager.GetOppositeLayoutWord(term),
							"fields": searchFields,
						},
					},
				},
				"must_not": []map[string]any{
					{
						"range": map[string]any{
							"count_in_stock": map[string]any{
								"lt": 1,
							},
						},
					},
				},
			},
		},
	}

	dataBytes, err := serializer.Marshal(&shouldQuery)
	if err != nil {
		return nil, tracing.TraceWithErr(span, errors.Wrap(err, "serializer.Marshal"))
	}

	e.log.Debugf("Search json query: %+v", shouldQuery)

	response, err := e.esClient.Search(
		e.esClient.Search.WithContext(ctx),
		e.esClient.Search.WithIndex(e.cfg.ElasticIndexes.ProductsIndex.Alias),
		e.esClient.Search.WithBody(bufio.NewReader(bytes.NewReader(dataBytes))),
		e.esClient.Search.WithPretty(),
		e.esClient.Search.WithHuman(),
		e.esClient.Search.WithTimeout(searchTimeout),
		e.esClient.Search.WithSize(pagination.GetSize()),
		e.esClient.Search.WithFrom(pagination.GetOffset()),
	)
	if err != nil {
		return nil, tracing.TraceWithErr(span, errors.Wrap(err, "esClient.Search"))
	}
	defer response.Body.Close()

	if response.IsError() {
		return nil, tracing.TraceWithErr(span, errors.Wrap(errors.New(response.String()), "esClient.Search error"))
	}

	e.log.Infof("repository search result: %s", response.String())

	hits := esclient.EsHits[*domain.Product]{}
	if err := serializer.NewDecoder(response.Body).Decode(&hits); err != nil {
		return nil, tracing.TraceWithErr(span, errors.Wrap(err, "serializer.Decode"))
	}

	responseList := make([]*domain.Product, len(hits.Hits.Hits))
	for i, source := range hits.Hits.Hits {
		responseList[i] = source.Source
	}

	e.log.Infof("repository search result responseList: %+v", responseList)
	return &domain.ProductSearch{
		List:               responseList,
		PaginationResponse: utils.NewPaginationResponse(hits.Hits.Total.Value, pagination),
	}, nil
}
