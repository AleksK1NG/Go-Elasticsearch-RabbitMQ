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
	"github.com/elastic/go-elasticsearch/v8"
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

func (e *esRepository) Search(ctx context.Context, term string) (any, error) {
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

	//mm := map[string]esclient.MultiMatch{"multi_match": esclient.MultiMatch{
	//	Query:  term,
	//	Fields: []string{"title"},
	//}}
	//
	//query := esclient.MultiMatchQuery{
	//	Query: esclient.Query{
	//		Bool: esclient.Bool{
	//			Must: []any{mm},
	//		},
	//	},
	//}

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
	)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.IsError() {
		return nil, errors.Wrap(errors.New(response.String()), "esClient.Search error")
	}

	e.log.Infof("repository search result: %s", response.String())

	hits := esclient.EsHits[domain.Product]{}
	err = json.NewDecoder(response.Body).Decode(&hits)
	if err != nil {
		return nil, err
	}

	responseList := make([]domain.Product, len(hits.Hits.Hits))
	for i, source := range hits.Hits.Hits {
		responseList[i] = source.Source
	}

	e.log.Infof("repository search result responseList: %+v", responseList)
	return responseList, nil
}

//func (e *esRepository) BulkIndex(ctx context.Context, product domain.Product) error {
//
//	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
//		NumWorkers: 10,
//		//FlushBytes:          0,
//		FlushInterval: 5 * time.Second,
//		Client:        e.esClient,
//		OnError: func(ctx context.Context, err error) {
//			e.log.Errorf("bulk indexer onError: %v", err)
//		},
//		OnFlushStart: func(ctx context.Context) context.Context {
//			e.log.Infof("starting bulk indexer for index: %s", e.cfg.ElasticIndexes.ProductsIndex.Name)
//			return ctx
//		},
//		OnFlushEnd: func(ctx context.Context) {
//			e.log.Infof("finished bulk indexer for index: %s", e.cfg.ElasticIndexes.ProductsIndex.Name)
//		},
//		Index:   e.cfg.ElasticIndexes.ProductsIndex.Name,
//		Human:   true,
//		Pretty:  true,
//		Timeout: 5 * time.Second,
//	})
//	if err != nil {
//		return errors.Wrap(err, "esutil.NewBulkIndexer")
//	}
//
//	dataBytes, err := json.Marshal(&product)
//	if err != nil {
//		return errors.Wrap(err, "json.Marshal")
//	}
//	if err := bulkIndexer.Add(ctx, esutil.BulkIndexerItem{
//		Index:      e.cfg.ElasticIndexes.ProductsIndex.Name,
//		Action:     "create",
//		DocumentID: product.ID,
//		Body:       bytes.NewReader(dataBytes),
//		OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, item2 esutil.BulkIndexerResponseItem) {
//			e.log.Infof("bulk add success item: %s, response: %s", item.DocumentID, item2.DocumentID)
//		},
//		OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, item2 esutil.BulkIndexerResponseItem, err error) {
//			e.log.Infof("bulk add err item: %s, response: %s, err: %v", item.DocumentID, item2.DocumentID, err)
//		},
//	}); err != nil {
//		return errors.Wrap(err, "bulkIndexer.Add")
//	}
//
//	response, err := e.esClient.Bulk(
//		bytes.NewReader(dataBytes),
//		e.esClient.Bulk.WithIndex(e.cfg.ElasticIndexes.ProductsIndex.Name),
//		e.esClient.Bulk.WithContext(ctx),
//		e.esClient.Bulk.WithPretty(),
//		e.esClient.Bulk.WithHuman(),
//		e.esClient.Bulk.WithTimeout(5*time.Second),
//	)
//	if err != nil {
//		return errors.Wrap(err, "esClient.Bulk")
//	}
//	defer response.Body.Close()
//
//	if response.IsError() {
//		return errors.Wrap(errors.New(response.String()), "esClient.Bulk response error")
//	}
//
//	e.log.Infof("document bulk indexed: %s", response.String())
//	return nil
//}
