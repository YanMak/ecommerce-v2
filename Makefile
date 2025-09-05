# ---- migrate config ----
# Путь к migrate CLI. Если не найден — бросим ошибку.
MIGRATE ?= $(shell command -v migrate 2>/dev/null)
MIGRATE_CMD = $(if $(MIGRATE),$(MIGRATE),$(error "migrate not found; install: go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest"))

# Базовый PG URL без имени БД, чтобы удобнее перегружать окружением.
PG_URL ?= postgres://postgres:postgres@localhost:5432

# Подключения для сервисов (можешь переопределять через env).
CRM_DB_URL   ?= $(PG_URL)/crm?sslmode=disable
ITEMS_DB_URL ?= $(PG_URL)/items?sslmode=disable

# Папки с миграциями (проверь, что они существуют).
CRM_MIG_DIR   ?= services/crm/db/migrations
ITEMS_MIG_DIR ?= services/items/db/migrations

.PHONY: migrate-up migrate-down migrate-version \
        migrate-crm-up migrate-crm-down migrate-crm-force migrate-crm-version migrate-crm-create \
        migrate-items-up migrate-items-down migrate-items-force migrate-items-version migrate-items-create

## ----- CRM -----
migrate-crm-up:
	$(MIGRATE_CMD) -path $(CRM_MIG_DIR) -database "$(CRM_DB_URL)" up

# N шагов вниз: make migrate-crm-down n=1 (по умолчанию 1)
migrate-crm-down:
	$(MIGRATE_CMD) -path $(CRM_MIG_DIR) -database "$(CRM_DB_URL)" down $(if $(n),$(n),1)

# Зафиксировать версию при кривом состоянии: make migrate-crm-force v=3
migrate-crm-force:
	@[ -n "$(v)" ] || (echo "Usage: make migrate-crm-force v=<version>"; exit 1)
	$(MIGRATE_CMD) -path $(CRM_MIG_DIR) -database "$(CRM_DB_URL)" force $(v)

migrate-crm-version:
	-$(MIGRATE_CMD) -path $(CRM_MIG_DIR) -database "$(CRM_DB_URL)" version || true

# Создать новый файл миграции: make migrate-crm-create name=init_tables
migrate-crm-create:
	@[ -n "$(name)" ] || (echo "Usage: make migrate-crm-create name=<snake_case>"; exit 1)
	mkdir -p $(CRM_MIG_DIR)
	$(MIGRATE_CMD) create -ext sql -dir $(CRM_MIG_DIR) -seq $(name)

## ----- ITEMS -----
migrate-items-up:
	$(MIGRATE_CMD) -path $(ITEMS_MIG_DIR) -database "$(ITEMS_DB_URL)" up

migrate-items-down:
	$(MIGRATE_CMD) -path $(ITEMS_MIG_DIR) -database "$(ITEMS_DB_URL)" down $(if $(n),$(n),1)

migrate-items-force:
	@[ -n "$(v)" ] || (echo "Usage: make migrate-items-force v=<version>"; exit 1)
	$(MIGRATE_CMD) -path $(ITEMS_MIG_DIR) -database "$(ITEMS_DB_URL)" force $(v)

migrate-items-version:
	-$(MIGRATE_CMD) -path $(ITEMS_MIG_DIR) -database "$(ITEMS_DB_URL)" version || true

migrate-items-create:
	@[ -n "$(name)" ] || (echo "Usage: make migrate-items-create name=<snake_case>"; exit 1)
	mkdir -p $(ITEMS_MIG_DIR)
	$(MIGRATE_CMD) create -ext sql -dir $(ITEMS_MIG_DIR) -seq $(name)

## ----- Convenience -----
# Запустить все апы для обоих сервисов
migrate-up: migrate-crm-up migrate-items-up

# Сначала откатываем items, затем crm (часто безопаснее)
migrate-down:
	$(MAKE) migrate-items-down n=$(if $(n),$(n),1)
	$(MAKE) migrate-crm-down n=$(if $(n),$(n),1)

# Показать версии обоих
migrate-version:
	@echo "CRM version:";   $(MAKE) -s migrate-crm-version
	@echo "ITEMS version:"; $(MAKE) -s migrate-items-version
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