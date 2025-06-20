.PHONY: build up down migrate start logs restart

run-d:
	docker-compose up -d --build

run:
	docker-compose up --build

down:
	docker-compose down

down-v:
	docker-compose down -v

start:
	docker-compose up surf_bot

logs:
	docker-compose logs -f

restart:
	docker-compose restart surf_bot

dbshell:
	docker exec -it surf_db psql -U app -d app

reset:
	docker-compose down -v && docker-compose up -d && make migrate

migrate:
	docker-compose run --rm surf_bot migrate

format:
	goimports -w .
	gofmt -w .

lint:
	golangci-lint run

check: format lint

mm:
ifndef name
	$(error ❌ No name provided, run for example: make mm create_users)
endif
	timestamp=$$(date +%Y%m%d%H%M%S); \
	filename="migrations/$${timestamp}_$(name).sql"; \
	touch $$filename; \
	echo "-- +goose Up\n\n\n-- +goose Down\n" > $$filename; \
	echo "✅  Migration created: $$filename"