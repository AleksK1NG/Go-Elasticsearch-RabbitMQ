package app

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/metrics"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/delivery/http/v1"
	productRabbitConsumer "github.com/AleksK1NG/go-elasticsearch/internal/product/delivery/rabbitmq"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/repository"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/usecase"
	"github.com/AleksK1NG/go-elasticsearch/pkg/esclient"
	"github.com/AleksK1NG/go-elasticsearch/pkg/keyboard_manager"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/middlewares"
	"github.com/AleksK1NG/go-elasticsearch/pkg/rabbitmq"
	"github.com/AleksK1NG/go-elasticsearch/pkg/tracing"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	amqp "github.com/rabbitmq/amqp091-go"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	waitShotDownDuration = 3 * time.Second
)

type app struct {
	log                   logger.Logger
	cfg                   *config.Config
	doneCh                chan struct{}
	elasticClient         *elasticsearch.Client
	echo                  *echo.Echo
	validate              *validator.Validate
	middlewareManager     middlewares.MiddlewareManager
	amqpConn              *amqp.Connection
	amqpChan              *amqp.Channel
	amqpPublisher         rabbitmq.AmqpPublisher
	keyboardLayoutManager keyboard_manager.KeyboardLayoutManager
	metricsServer         *echo.Echo
	healthCheckServer     *http.Server
	metrics               *metrics.SearchMicroserviceMetrics
}

func NewApp(log logger.Logger, cfg *config.Config) *app {
	return &app{log: log, cfg: cfg, validate: validator.New(), doneCh: make(chan struct{}), echo: echo.New()}
}

func (a *app) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	if err := a.loadKeyMappings(); err != nil {
		return err
	}

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
	a.metrics = metrics.NewSearchMicroserviceMetrics(a.cfg)

	if err := a.initRabbitMQ(ctx); err != nil {
		return err
	}
	defer a.closeRabbitConn()

	queue, err := rabbitmq.DeclareBinding(ctx, a.amqpChan, a.cfg.ExchangeAndQueueBindings.IndexProductBinding)
	if err != nil {
		return err
	}
	a.log.Infof("rabbitmq queue created: %+v for binding: %+v", queue, a.cfg.ExchangeAndQueueBindings.IndexProductBinding)

	if err := a.initRabbitMQPublisher(ctx); err != nil {
		return err
	}
	defer a.amqpPublisher.Close()

	if err := a.initElasticsearchClient(ctx); err != nil {
		return err
	}

	// connect elastic
	elasticInfoResponse, err := esclient.Info(ctx, a.elasticClient)
	if err != nil {
		return err
	}
	a.log.Infof("Elastic info response: %s", elasticInfoResponse.String())

	if err := a.initIndexes(ctx); err != nil {
		return err
	}

	elasticRepository := repository.NewEsRepository(a.log, a.cfg, a.elasticClient, a.keyboardLayoutManager)
	productUseCase := usecase.NewProductUseCase(a.log, a.cfg, elasticRepository, a.amqpPublisher)
	productController := v1.NewProductController(a.log, a.cfg, productUseCase, a.echo.Group(a.cfg.Http.ProductsPath), a.validate, a.metrics)
	productController.MapRoutes()

	go func() {
		if err := a.runHttpServer(); err != nil {
			a.log.Errorf("(runHttpServer) err: %v", err)
			cancel()
		}
	}()
	a.log.Infof("%s is listening on PORT: %v", GetMicroserviceName(a.cfg), a.cfg.Http.Port)

	productConsumer := productRabbitConsumer.NewConsumer(a.log, a.cfg, a.amqpConn, a.amqpChan, productUseCase, a.elasticClient, a.metrics)
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
			a.log.Errorf("(rabbitmq.ConsumeQueue) err: %v", err)
			cancel()
		}
	}()

	a.runMetrics(cancel)
	a.runHealthCheck(ctx)

	<-ctx.Done()
	a.waitShootDown(waitShotDownDuration)

	if err := a.echo.Shutdown(ctx); err != nil {
		a.log.Warnf("(Shutdown) err: %v", err)
	}
	if err := a.metricsServer.Shutdown(ctx); err != nil {
		a.log.Warnf("(Shutdown) metricsServer err: %v", err)
	}
	if err := a.shutDownHealthCheckServer(ctx); err != nil {
		a.log.Warnf("(shutDownHealthCheckServer) HealthCheckServer err: %v", err)
	}

	<-a.doneCh
	a.log.Infof("%s app exited properly", GetMicroserviceName(a.cfg))
	return nil
}
