package app

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/pkg/constants"
	"github.com/AleksK1NG/go-elasticsearch/pkg/esclient"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

func (a *app) runHealthCheck(ctx context.Context) {
	health := healthcheck.NewHandler()

	mux := http.NewServeMux()
	mux.HandleFunc(a.cfg.Probes.LivenessPath, health.LiveEndpoint)
	mux.HandleFunc(a.cfg.Probes.ReadinessPath, health.ReadyEndpoint)

	a.healthCheckServer = &http.Server{
		Handler:      mux,
		Addr:         a.cfg.Probes.Port,
		WriteTimeout: writeTimeout,
		ReadTimeout:  readTimeout,
	}

	a.configureHealthCheckEndpoints(ctx, health)

	go func() {
		a.log.Infof("(%s) Kubernetes probes listening on port: %s", a.cfg.ServiceName, a.cfg.Probes.Port)
		if err := a.healthCheckServer.ListenAndServe(); err != nil {
			a.log.Errorf("(ListenAndServe) err: %v", err)
		}
	}()
}

func (a *app) configureHealthCheckEndpoints(ctx context.Context, health healthcheck.Handler) {

	health.AddReadinessCheck(constants.ElasticSearch, healthcheck.AsyncWithContext(ctx, func() error {
		_, err := esclient.Info(ctx, a.elasticClient)
		if err != nil {
			a.log.Warnf("(ElasticSearch Readiness Check) err: %v", err)
			return errors.Wrap(err, "esClient.Info")
		}
		return nil
	}, time.Duration(a.cfg.Probes.CheckIntervalSeconds)*time.Second))

	health.AddLivenessCheck(constants.ElasticSearch, healthcheck.AsyncWithContext(ctx, func() error {
		_, err := esclient.Info(ctx, a.elasticClient)
		if err != nil {
			a.log.Warnf("(ElasticSearch Liveness Check) err: %v", err)
			return errors.Wrap(err, "esClient.Info")
		}
		return nil
	}, time.Duration(a.cfg.Probes.CheckIntervalSeconds)*time.Second))

	health.AddLivenessCheck(constants.RabbitMQ, healthcheck.AsyncWithContext(ctx, func() error {
		if a.amqpConn.IsClosed() || a.amqpChan.IsClosed() {
			a.log.Warnf("(rabbitmq Liveness Check) err: %v", errors.New("amqp conn closed"))
			return errors.New("amqp conn closed")
		}
		return nil
	}, time.Duration(a.cfg.Probes.CheckIntervalSeconds)*time.Second))

	health.AddReadinessCheck(constants.RabbitMQ, healthcheck.AsyncWithContext(ctx, func() error {
		if a.amqpConn.IsClosed() || a.amqpChan.IsClosed() {
			a.log.Warnf("(rabbitmq Readiness Check) err: %v", errors.New("amqp conn closed"))
			return errors.New("amqp conn closed")
		}
		return nil
	}, time.Duration(a.cfg.Probes.CheckIntervalSeconds)*time.Second))
}

func (a *app) shutDownHealthCheckServer(ctx context.Context) error {
	return a.healthCheckServer.Shutdown(ctx)
}
