.PHONY:

# ==============================================================================
# Docker

local:
	@echo Clearing elasticserach data
	rm -rf ./es-data01
	@echo Clearing prometheus data
	rm -rf ./prometheus
	@echo Starting local docker compose
	docker-compose -f docker-compose.local.yaml up -d --build

develop:
	@echo Clearing elasticserach data
	rm -rf ./es-data01
	@echo Clearing prometheus data
	rm -rf ./prometheus
	@echo Starting local docker compose
	docker-compose -f docker-compose.yaml up -d --build

upload:
	docker build -t alexanderbryksin/search_microservice:latest -f ./Dockerfile .
	docker push alexanderbryksin/search_microservice:latest

# ==============================================================================
# Docker support

FILES := $(shell docker ps -aq)

down-local:
	docker stop $(FILES)
	docker rm $(FILES)

clean:
	docker system prune -f

logs-local:
	docker logs -f $(FILES)


# ==============================================================================
# k8s support

helm_install:
	helm install -f k8s/microservice/values.yaml search k8s/microservice

helm_uninstall:
	helm uninstall search

dry_run:
	kubectl apply --dry-run=client -f k8s/microservice/templates/servicemonitor.yaml

port_forward_microservice:
	kubectl port-forward services/microservice-service 8000:8000

port_forward_kibana:
	kubectl port-forward services/kibana 5601:5601

port_forward_rabbitmq:
	kubectl port-forward services/rabbitmq 15672:15672

port_forward_prometheus:
	kubectl port-forward services/prometheus-kube-prometheus-prometheus 9090:9090

minikube_start:
	minikube start --memory 16000 --cpus 4


prometheus_install:
	helm install prometheus prometheus-community/kube-prometheus-stack

prometheus_install:
	helm uninstall prometheus