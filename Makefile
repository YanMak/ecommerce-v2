# ===== Common =====
.PHONY: help create up down redo force version sqlc schema dump env

# какой сервис дергаем (по умолчанию crm)
SERVICE ?= crm

# корневая папка сервиса
SVC_DIR  := services/$(SERVICE)

# миграции/схема/queries
DB_DIR    := $(SVC_DIR)/db
MIGR_DIR  := $(DB_DIR)/migrations
SCHEMA    := $(DB_DIR)/schema.sql

# DSN базы можно переопределить через окружение
# пример: DB_URL=postgres://user:pass@localhost:5432/rosk?sslmode=disable
DB_URL ?= postgres://postgres:postgres@localhost:5432/$(SERVICE)?sslmode=disable

# ===== Migrate (golang-migrate) =====
create:
	@test $(name)
	migrate create -ext sql -dir $(MIGR_DIR) -seq $(name)

up:
	migrate -path $(MIGR_DIR) -database "$(DB_URL)" up

down:
	migrate -path $(MIGR_DIR) -database "$(DB_URL)" down 1

redo:
	migrate -path $(MIGR_DIR) -database "$(DB_URL)" down 1
	migrate -path $(MIGR_DIR) -database "$(DB_URL)" up 1

force:
	@test $(version)
	migrate -path $(MIGR_DIR) -database "$(DB_URL)" force $(version)

version:
	migrate -path $(MIGR_DIR) -database "$(DB_URL)" version

# ===== sqlc =====
sqlc:
	cd $(SVC_DIR) && sqlc generate

# ===== schema.sql из живой БД (удобно держать sqlc на актуальной схеме) =====
# Требуется установленный pg_dump
schema:
	pg_dump "$(DB_URL)" --schema-only --no-owner --no-privileges > "$(SCHEMA)"

# ===== Вспомогалки =====
help:
	@echo "Usage: make [target] SERVICE=crm DB_URL=postgres://..."
	@echo
	@echo "Migrate:"
	@echo "  make create name=<snake_case>   # создать пустую миграцию"
	@echo "  make up                          # применить все up"
	@echo "  make down                        # откатить одну миграцию"
	@echo "  make redo                        # down 1 && up 1"
	@echo "  make force version=<n>           # принудительно выставить версию"
	@echo "  make version                     # показать версию схеме"
	@echo
	@echo "SQLC:"
	@echo "  make sqlc                        # sqlc generate (читает $(SVC_DIR)/sqlc.yaml)"
	@echo
	@echo "Schema:"
	@echo "  make schema                      # pg_dump -> $(SCHEMA)"

#protoc -I api \
  --go_out=./gen --go_opt=paths=source_relative \
  --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
  api/inventory/v1/stock.proto \
  api/inventory/v1/stock_admin.proto \
  api/catalog/v1/items_admin.proto \
  api/catalog/v1/catalog_read.proto