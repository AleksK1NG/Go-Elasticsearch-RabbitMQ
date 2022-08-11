package metrics

import (
	"fmt"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type SearchMicroserviceMetrics struct {
	SuccessHttpRequests prometheus.Counter
	ErrorHttpRequests   prometheus.Counter

	HttpSuccessIndexAsyncRequests prometheus.Counter
	HttpSuccessIndexRequests      prometheus.Counter
	HttpSuccessSearchRequests     prometheus.Counter

	RabbitMQSuccessBatchInsertMessages prometheus.Counter
}

func NewSearchMicroserviceMetrics(cfg *config.Config) *SearchMicroserviceMetrics {
	return &SearchMicroserviceMetrics{

		SuccessHttpRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_success_http_requests_total", cfg.ServiceName),
			Help: "The total number of success http requests",
		}),
		ErrorHttpRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_error_http_requests_total", cfg.ServiceName),
			Help: "The total number of error http requests",
		}),
		HttpSuccessIndexRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_success_http_index_requests_total", cfg.ServiceName),
			Help: "The total number of success_http_index_requests_total requests",
		}),
		HttpSuccessIndexAsyncRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_success_http_index_async_requests_total", cfg.ServiceName),
			Help: "The total number of success_http_index_async_requests_total requests",
		}),
		HttpSuccessSearchRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_success_http_search_requests_total", cfg.ServiceName),
			Help: "The total number of success_http_search_requests_total requests",
		}),
		RabbitMQSuccessBatchInsertMessages: promauto.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_success_rabbitmq_batch_insert_messages_total", cfg.ServiceName),
			Help: "The total number of success_rabbitmq_batch_insert_messages requests",
		}),
	}
}
