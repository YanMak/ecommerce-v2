package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	invpb "github.com/YanMak/ecommerce/v2/gen/inventory/v1"
	grpcstock "github.com/YanMak/ecommerce/v2/services/inventory-svc/internal/adapters/inbound/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// ---- Временная заглушка под порт приложения ----
// Заменишь на свою реализацию InventoryQueries из слоя app.
type dummyQueries struct{}

func (d *dummyQueries) GetStock(ctx context.Context, itemID int64, locationCode string) (grpcstock.StockDTO, error) {
	return grpcstock.StockDTO{
		ItemID:    itemID,
		Available: 0,
		Locations: nil,
		UpdatedAt: time.Now(),
	}, nil
}

func (d *dummyQueries) BatchGetStock(ctx context.Context, itemIDs []int64, locationCode string) ([]grpcstock.StockDTO, error) {
	out := make([]grpcstock.StockDTO, 0, len(itemIDs))
	now := time.Now()
	for _, id := range itemIDs {
		out = append(out, grpcstock.StockDTO{
			ItemID:    id,
			Available: 0,
			Locations: nil,
			UpdatedAt: now,
		})
	}
	return out, nil
}

func main() {
	// Ctrl+C / SIGTERM -> корректная остановка
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := ":8081"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}

	grpcSrv := grpc.NewServer()
	// ВАЖНО: используем правильный конструктор и реальную (или заглушечную) реализацию порта
	invpb.RegisterStockServiceServer(grpcSrv, grpcstock.NewServer(&dummyQueries{}))

	// Удобно для grpcurl / отладки
	reflection.Register(grpcSrv)

	<-ctx.Done()
	log.Println("shutting down gracefully...")
	grpcSrv.GracefulStop()
	_ = lis.Close()
}
