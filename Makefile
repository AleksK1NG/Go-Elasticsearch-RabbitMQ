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