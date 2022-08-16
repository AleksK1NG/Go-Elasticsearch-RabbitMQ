Go ElasticSearch and RabbitMQ full-text search microservice 👋✨💫

#### 👨‍💻 Full list what has been used:
[Elasticsearch](https://github.com/elastic/go-elasticsearch) client for Go <br/>
[RabbitMQ](https://github.com/rabbitmq/amqp091-go) Go RabbitMQ Client Library <br/>
[Jaeger](https://www.jaegertracing.io/) open source, end-to-end distributed [tracing](https://opentracing.io/) <br/>
[Prometheus](https://prometheus.io/) monitoring and alerting <br/>
[Grafana](https://grafana.com/) for to compose observability dashboards with everything from Prometheus <br/>
[Echo](https://github.com/labstack/echo) web framework <br/>
[Kibana](https://github.com/labstack/echo) is user interface that lets you visualize your Elasticsearch <br/>
[Docker](https://www.docker.com/) and docker-compose <br/>
[Kubernetes](https://kubernetes.io/) K8s <br/>
[Helm](https://helm.sh/) The package manager for Kubernetes <br/>


### RabbitMQ UI:

http://localhost:15672

### Jaeger UI:

http://localhost:16686

### Prometheus UI:

http://localhost:9090

### Grafana UI:

http://localhost:3005

### Kibana UI:

http://localhost:5601/app/home#/


For local development 🙌👨‍💻🚀:

```
make local // for run docker compose
make run_es // run microservice
```
or
```
make develop // run all in docker compose
```

for k8s
```
make install_all // helm install
```