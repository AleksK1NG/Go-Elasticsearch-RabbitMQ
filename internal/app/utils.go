package app

import (
	"context"
	"fmt"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/pkg/esclient"
	"github.com/AleksK1NG/go-elasticsearch/pkg/keyboard_manager"
	"github.com/AleksK1NG/go-elasticsearch/pkg/middlewares"
	"github.com/AleksK1NG/go-elasticsearch/pkg/serializer"
	"github.com/pkg/errors"
	"io"
	"os"
	"strings"
	"time"
)

func (a *app) waitShootDown(duration time.Duration) {
	go func() {
		time.Sleep(duration)
		a.doneCh <- struct{}{}
	}()
}

func GetMicroserviceName(cfg *config.Config) string {
	return fmt.Sprintf("(%s)", strings.ToUpper(cfg.ServiceName))
}

func (a *app) initIndexes(ctx context.Context) error {
	exists, err := a.isIndexExists(ctx, a.cfg.ElasticIndexes.ProductsIndex.Name)
	if err != nil {
		return err
	}
	if !exists {
		if err := a.uploadElasticMappings(ctx, a.cfg.ElasticIndexes.ProductsIndex); err != nil {
			return err
		}
	}
	a.log.Infof("index exists: %+v", a.cfg.ElasticIndexes.ProductsIndex)

	response, err := a.elasticClient.Indices.PutAlias(
		[]string{a.cfg.ElasticIndexes.ProductsIndex.Name},
		a.cfg.ElasticIndexes.ProductsIndex.Alias,
		a.elasticClient.Indices.PutAlias.WithContext(ctx),
		a.elasticClient.Indices.PutAlias.WithHuman(),
		a.elasticClient.Indices.PutAlias.WithPretty(),
		a.elasticClient.Indices.PutAlias.WithTimeout(5*time.Second),
	)
	if err != nil {
		a.log.Errorf("Indices.PutAlias err: %v", err)
		return err
	}
	defer response.Body.Close()

	if response.IsError() {
		a.log.Errorf("create alias error: %s", response.String())
		return errors.Wrap(errors.New(response.String()), "response.IsError")
	}

	a.log.Infof("alias: %s for indexes: %+v is created", a.cfg.ElasticIndexes.ProductsIndex.Alias, []string{a.cfg.ElasticIndexes.ProductsIndex.Name})
	return nil
}

func (a *app) isIndexExists(ctx context.Context, indexName string) (bool, error) {
	response, err := esclient.Exists(ctx, a.elasticClient, []string{indexName})
	if err != nil {
		a.log.Errorf("initIndexes err: %v", err)
		return false, errors.Wrap(err, "esclient.Exists")
	}
	defer response.Body.Close()

	if response.IsError() && response.StatusCode == 404 {
		return false, nil
	}

	a.log.Infof("exists response: %s", response)
	return true, nil
}

func (a *app) uploadElasticMappings(ctx context.Context, indexConfig esclient.ElasticIndex) error {
	getwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "os.Getwd")
	}
	path := fmt.Sprintf("%s/%s", getwd, indexConfig.Path)

	mappingsFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer mappingsFile.Close()

	mappingsBytes, err := io.ReadAll(mappingsFile)
	if err != nil {
		return err
	}

	a.log.Infof("loaded mappings bytes: %s", string(mappingsBytes))

	response, err := esclient.CreateIndex(ctx, a.elasticClient, indexConfig.Name, mappingsBytes)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.IsError() && response.StatusCode != 400 {
		return errors.New(fmt.Sprintf("err init index: %s", response.String()))
	}

	a.log.Infof("created index: %s", response.String())
	return nil
}

func (a *app) loadKeyboardLayoutManager() (keyboard_manager.KeyboardLayoutManager, error) {
	getwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "os.Getwd")
	}

	keysJsonPathFile, err := os.Open(fmt.Sprintf("%s/config/translate.json", getwd))
	if err != nil {
		return nil, err
	}
	defer keysJsonPathFile.Close()

	keysJsonBytes, err := io.ReadAll(keysJsonPathFile)
	if err != nil {
		return nil, err
	}

	a.log.Infof("keys mappings: %s", string(keysJsonBytes))

	keyMappings := map[string]string{}
	if err := serializer.Unmarshal(keysJsonBytes, &keyMappings); err != nil {
		return nil, err
	}
	a.log.Infof("keys mappings data: %+v", keyMappings)

	keyboardLayoutManager := keyboard_manager.NewKeyboardLayoutManager(a.log, keyMappings)

	a.log.Infof("keys mappings processed data: %s", keyboardLayoutManager.GetOppositeLayoutWord("Фдуч ЗКЩ"))

	return keyboardLayoutManager, nil
}

func (a *app) getHttpMetricsCb() middlewares.MiddlewareMetricsCb {
	return func(err error) {
		if err != nil {
			a.metrics.ErrorHttpRequests.Inc()
		} else {
			a.metrics.SuccessHttpRequests.Inc()
		}
	}
}
