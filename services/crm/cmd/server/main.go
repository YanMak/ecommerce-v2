package main

import (
	"context"
	"fmt"
	"log"
	"net"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	"github.com/YanMak/ecommerce/v2/pkg/grpcx"
	grpcin "github.com/YanMak/ecommerce/v2/services/crm/internal/adapters/inbound/grpc"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

func runGRPC(addr string, pool *pgxpool.Pool) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcx.UnaryServerMetaInterceptor,
			// тут позже можно добавить лог/метрики/рековери-интерсепторы
		),
		grpc.ChainStreamInterceptor(
			grpcx.StreamServerMetaInterceptor, // если будут streaming RPC
		),
	)
	uc := usecase.NewCertificatesUC(pool)
	crmpb.RegisterCertificatesServer(s, grpcin.NewCertificatesServer(uc))

	return s.Serve(lis)
}

func main() {

	dsn := "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable"

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("pgxpool:", err)
	}
	defer pool.Close()

	runGRPC(":50051", pool)

	fmt.Println("hallo")
}
