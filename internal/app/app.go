package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/repository"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/transport/http/v1"
	productRabbitConsumer "github.com/AleksK1NG/go-elasticsearch/internal/product/transport/rabbitmq"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/usecase"
	"github.com/AleksK1NG/go-elasticsearch/pkg/elastic"
	"github.com/AleksK1NG/go-elasticsearch/pkg/esclient"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/middlewares"
	"github.com/AleksK1NG/go-elasticsearch/pkg/misstype_manager"
	"github.com/AleksK1NG/go-elasticsearch/pkg/rabbitmq"
	"github.com/AleksK1NG/go-elasticsearch/pkg/tracing"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	waitShotDownDuration = 3 * time.Second
)

type app struct {
	log               logger.Logger
	cfg               *config.Config
	doneCh            chan struct{}
	elasticClient     *elasticsearch.Client
	echo              *echo.Echo
	validate          *validator.Validate
	middlewareManager middlewares.MiddlewareManager
	amqpConn          *amqp.Connection
	amqpChan          *amqp.Channel
	amqpPublisher     rabbitmq.AmqpPublisher
	missTypeManager   misstype_manager.MissTypeManager
}

func NewApp(log logger.Logger, cfg *config.Config) *app {
	return &app{log: log, cfg: cfg, validate: validator.New(), doneCh: make(chan struct{}), echo: echo.New()}
}

func (a *app) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	missTypeManager, err := a.loadKeysMappings()
	if err != nil {
		return err
	}
	a.missTypeManager = missTypeManager

	// enable tracing
	if a.cfg.Jaeger.Enable {
		tracer, closer, err := tracing.NewJaegerTracer(a.cfg.Jaeger)
		if err != nil {
			return err
		}
		defer closer.Close() // nolint: errcheck
		opentracing.SetGlobalTracer(tracer)
	}

	a.middlewareManager = middlewares.NewMiddlewareManager(a.log, a.cfg, a.getHttpMetricsCb())

	amqpConn, err := rabbitmq.NewRabbitMQConnection(a.cfg.RabbitMQ)
	if err != nil {
		return err
	}
	defer amqpConn.Close()
	a.amqpConn = amqpConn

	amqpChan, err := amqpConn.Channel()
	if err != nil {
		return err
	}
	defer amqpChan.Close()
	a.amqpChan = amqpChan

	if err := a.amqpChan.Qos(1, 0, true); err != nil {
		return err
	}

	a.log.Infof("rabbitmq connected: %+v", a.amqpConn)

	queue, err := rabbitmq.DeclareBinding(ctx, a.amqpChan, a.cfg.ExchangeAndQueueBindings.IndexProductBinding)
	if err != nil {
		return err
	}
	a.log.Infof("rabbitmq queue created: %+v for binding: %+v", queue, a.cfg.ExchangeAndQueueBindings.IndexProductBinding)

	elasticSearchClient, err := elastic.NewElasticSearchClient(a.cfg.ElasticSearch)
	if err != nil {
		return err
	}
	a.elasticClient = elasticSearchClient

	// connect elastic
	elasticInfoResponse, err := esclient.Info(ctx, a.elasticClient)
	if err != nil {
		return err
	}
	a.log.Infof("Elastic info response: %s", elasticInfoResponse.String())

	if err := a.initIndexes(ctx); err != nil {
		return err
	}

	elasticRepository := repository.NewEsRepository(a.log, a.cfg, a.elasticClient, a.missTypeManager)
	productUseCase := usecase.NewProductUseCase(a.log, a.cfg, elasticRepository)
	productController := v1.NewProductController(a.log, a.cfg, productUseCase, a.echo.Group(a.cfg.Http.ProductsPath), a.validate)
	productController.MapRoutes()

	go func() {
		if err := a.runHttpServer(); err != nil {
			a.log.Errorf("(runHttpServer) err: %v", err)
			cancel()
		}
	}()
	a.log.Infof("%s is listening on PORT: %v", GetMicroserviceName(a.cfg), a.cfg.Http.Port)

	productConsumer := productRabbitConsumer.NewConsumer(a.log, a.cfg, a.amqpConn, a.amqpChan, productUseCase, a.elasticClient)
	if err := productConsumer.InitBulkIndexer(); err != nil {
		a.log.Errorf("(InitBulkIndexer) err: %v", err)
		cancel()
	}
	defer productConsumer.Close(ctx)

	go func() {
		if err := rabbitmq.ConsumeQueue(
			ctx,
			a.amqpChan,
			a.cfg.ExchangeAndQueueBindings.IndexProductBinding.Concurrency,
			a.cfg.ExchangeAndQueueBindings.IndexProductBinding.QueueName,
			a.cfg.ExchangeAndQueueBindings.IndexProductBinding.Consumer,
			productConsumer.ConsumeIndexDeliveries,
		); err != nil {
			a.log.Errorf("")
			cancel()
		}
	}()

	a.amqpPublisher, err = rabbitmq.NewPublisher(a.cfg.RabbitMQ, a.log)
	if err != nil {
		return err
	}
	defer a.amqpPublisher.Close()

	<-ctx.Done()
	a.waitShootDown(waitShotDownDuration)

	if err := a.echo.Shutdown(ctx); err != nil {
		a.log.Warnf("(Shutdown) err: %v", err)
	}

	<-a.doneCh
	a.log.Infof("%s app exited properly", GetMicroserviceName(a.cfg))
	return nil
}

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

func (a *app) loadKeysMappings() (*misstype_manager.KeyboardMissTypeManager, error) {
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
	if err := json.Unmarshal(keysJsonBytes, &keyMappings); err != nil {
		return nil, err
	}
	a.log.Infof("keys mappings data: %+v", keyMappings)

	missTypeManager := misstype_manager.NewMissTypeManager(a.log, keyMappings)

	a.log.Infof("keys mappings processed data: %s", missTypeManager.GetMissTypedWord("Фдуч ЗКЩ"))

	return missTypeManager, nil
}

func (a *app) getHttpMetricsCb() middlewares.MiddlewareMetricsCb {
	return func(err error) {
		if err != nil {
			//a.metrics.ErrorHttpRequests.Inc()
		} else {
			//a.metrics.SuccessHttpRequests.Inc()
		}
	}
}
