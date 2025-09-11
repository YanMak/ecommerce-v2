package main

import (
	"log"
	"net/http"
	"os"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/adapters/inbound/httpapi"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/app/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcx "github.com/YanMak/ecommerce/v2/pkg/grpcx"
	prommetrics "github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"

	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"
)

func main() {

	// logger
	logger, err := tlog.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() //nolint:errcheck
	reqLog := logger.With(

		zap.String("service", "api-gateway"),
		zap.String("env", os.Getenv("ENV")),
		zap.String("version", "buildVersionXXX"),
	)

	// chi-маршрут для экспорта метрик
	reg, cols := prommetrics.New()

	// gRPC клиент CRM — создаём ОДИН раз, реиспользуем
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO: TLS позже
		grpc.WithChainUnaryInterceptor(
			grpcx.UnaryClientMetaInterceptor,           // прокидка request-id/idempotency
			grpcx.UnaryClientMetricsInterceptor(cols)), // ← метрики клиента
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	crmClient := crmpb.NewCertificatesClient(conn)
	certsUC := usecase.NewCertificatesUC(crmClient)

	srv := httpapi.NewServer(reqLog, reg, cols, certsUC)
	log.Println("HTTP listening on :8080")
	if err := http.ListenAndServe(":8080", srv); err != nil {
		log.Fatal(err)
	}

	// // HTTP router: middleware для request-id и handler, который вызывает usecase
	// mux := http.NewServeMux()
	// mux.Handle("/crm/certificates/search",
	// 	middleware.WithRequestID(
	// 		middleware.WithIdempotencyKey(
	// 			crmapi.ValidateCertsSearch(
	// 				crmhandlers.Search(certsUC)))))

	// mux.Handle("/crm/certificates/search/raw",
	// 	middleware.WithRequestID(
	// 		middleware.WithIdempotencyKey(
	// 			crmhandlers.Search(certsUC))))

	// mux.Handle("/healthz", middleware.WithRequestID(crmhandlers.Healtz(certsUC)))

	// log.Println("api-gateway listening on :8080")
	// log.Fatal(http.ListenAndServe(":8080", mux))
}

// func main_() {

// 	ctx := context.Background()
// 	ctx = tctx.WithRequestID(ctx, uuid.NewString()) //uuid.NewString()
// 	ctx = tctx.WithIdempotencyKey(ctx, "idempotency-key-454354-454-34543-4535")

// 	crmClient, err := clients.NewCRMClient(ctx, "localhost:50051")
// 	if err != nil {
// 		panic("cant create crm client")
// 	}

// 	req := &crmpb.SearchCertificatesRequest{
// 		Q: grpcx.S(ptr.To("Сертификат")),
// 		// …
// 		Page:      int32(0),
// 		PerPage:   int32(201),
// 		CreatedTo: grpcx.TS(ptr.To(time.Now())),
// 	}

// 	resp, err := crmClient.Certs.SearchCertificates(ctx, req)

// 	fmt.Printf("hallo, %+v\n %+v", resp, err)
// }
