.PHONY: up
up:
	docker compose up -d --build

.PHONY: down
down:
	docker compose down

.PHONY: logs
logs:
	docker compose logs -f

.PHONY: test
test:
	cd services/ads-service && go test ./...
	cd services/billing-service && go test ./...

.PHONY: wire
wire:
	cd services/ads-service && go generate ./internal/di
	cd services/billing-service && go generate ./internal/di
