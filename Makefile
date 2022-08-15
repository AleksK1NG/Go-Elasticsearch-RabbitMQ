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

install_all:
	@echo Updating helm repositories
	helm repo update
	@echo Creating monitoring namespace
	kubectl create namespace monitoring
	@echo Creating monitoring helm chart
	helm install monitoring prometheus-community/kube-prometheus-stack -n monitoring
	@echo Creating search microservice helm chart
	helm install -f k8s/microservice/values.yaml search k8s/microservice

helm_install:
	kubens default
	helm install -f k8s/microservice/values.yaml search k8s/microservice

uninstall_all:
	kubens monitoring
	helm uninstall monitoring
	kubens default
	helm uninstall search

dry_run:
	kubens default
	kubectl apply --dry-run=client -f k8s/microservice/templates/servicemonitor.yaml

port_forward_search_microservice:
	kubens default
	kubectl port-forward services/microservice-service 8000:8000

port_forward_kibana:
	kubens default
	kubectl port-forward services/kibana 5601:5601

port_forward_rabbitmq:
	kubens default
	kubectl port-forward services/rabbitmq 15672:15672

port_forward_prometheus:
	kubens monitoring
	kubectl port-forward services/monitoring-kube-prometheus-prometheus 9090:9090

minikube_start:
	minikube start --memory 16000 --cpus 4

monitoring_install:
	helm repo update
	kubectl create namespace monitoring
	helm install monitoring prometheus-community/kube-prometheus-stack -n monitoring

monitoring_uninstall:
	helm uninstall prometheus

port_forward_search_service_monitor:
	kubens monitoring
	kubectl port-forward services/monitoring-kube-prometheus-prometheus 9090:9090

apply_search_service_monitor:
	kubectl apply -f k8s/microservice/servicemonitor.yaml