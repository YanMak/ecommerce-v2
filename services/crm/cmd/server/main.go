package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"
	grpcin "github.com/YanMak/ecommerce/v2/services/crm/internal/adapters/inbound/grpc"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	grpcx "github.com/YanMak/ecommerce/v2/pkg/grpcx"

	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"

	crmmetrics "github.com/YanMak/ecommerce/v2/services/crm/internal/metrics"
)

func runGRPC(addr string, pool *pgxpool.Pool, cols *prom.Collectors, logger *zap.Logger, crmM *crmmetrics.CRM) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcx.UnaryServerMetaInterceptor,
			// тут позже можно добавить лог/метрики/рековери-интерсепторы
			grpcx.UnaryServerZapLogger(logger),
			grpcx.UnaryServerMetricsInterceptor(cols),
		),
		grpc.ChainStreamInterceptor(
			grpcx.StreamServerMetaInterceptor, // если будут streaming RPC
		),
	)
	uc := usecase.NewCertificatesUC(pool, crmM)
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

	//metrics
	reg, cols := prom.New()
	crmM := crmmetrics.Register(reg)

	base, _ := tlog.NewProduction()
	base = base.With(zap.String("service", "crm"), zap.String("env", os.Getenv("ENV")))

	go func() {
		mux := chi.NewRouter()
		mux.Method("GET", "/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		_ = http.ListenAndServe(":8081", mux)
	}()

	err = runGRPC(":50051", pool, cols, base, crmM)
	if err != nil {
		panic(err)
	}

	fmt.Println("hallo")
}
