protoc -I api \
  --go_out=./gen --go_opt=paths=source_relative \
  --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
  api/inventory/v1/stock.proto \
  api/inventory/v1/stock_admin.proto \
  api/catalog/v1/items_admin.proto \
  api/catalog/v1/catalog_read.proto