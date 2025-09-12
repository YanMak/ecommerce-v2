package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	crmhandlers "github.com/YanMak/ecommerce/v2/services/api-gateway/internal/adapters/inbound/httpapi/handlers/crm"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/app/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"

	grpcx "github.com/YanMak/ecommerce/v2/pkg/grpcx"
	prommetrics "github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"

	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/bind"
	httpmdw "github.com/YanMak/ecommerce/v2/pkg/httpx/middleware"

	cfg "github.com/YanMak/ecommerce/v2/pkg/config"

	gwdto "github.com/YanMak/ecommerce/v2/services/api-gateway/internal/adapters/inbound/httpapi/dto"
)

// getenv с дефолтом
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// Создаём TLS-креды клиента для подключения к CRM
func clientCredsToCRM() (grpc.DialOption, error) {
	caFile := getenv("CRM_TLS_CA_FILE", "/etc/enterprise/tls/ca/ca.pem")
	cliCertFile := getenv("GATEWAY_TLS_CERT_FILE", "/etc/enterprise/tls/gateway/gateway.pem")
	cliKeyFile := getenv("GATEWAY_TLS_KEY_FILE", "/etc/enterprise/tls/gateway/gateway.key")

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("append CA PEM failed")
	}
	// клиентский сертификат (для mTLS)
	cliCert, err := tls.LoadX509KeyPair(cliCertFile, cliKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load client cert: %w", err)
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    roots,
		// ServerName можно не указывать: Go возьмёт хост из адреса Dial (localhost или 10.0.12.20)
		Certificates: []tls.Certificate{cliCert},
	}
	if sni := os.Getenv("CRM_TLS_SERVER_NAME"); sni != "" {
		tlsCfg.ServerName = sni
	}
	return grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)), nil
}

func main() {
	// ---- config
	httpAddr := cfg.Str("HTTP_ADDR", ":8080")
	adminAddr := cfg.Str("ADMIN_ADDR", ":8088")
	readTO := cfg.Dur("HTTP_READ_TIMEOUT", 5*time.Second)
	writeTO := cfg.Dur("HTTP_WRITE_TIMEOUT", 10*time.Second)
	idleTO := cfg.Dur("HTTP_IDLE_TIMEOUT", 60*time.Second)
	shutdownTO := cfg.Dur("SHUTDOWN_TIMEOUT", 10*time.Second)

	// ---- telemetry
	reg, cols := prommetrics.New()
	logger, err := tlog.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() //nolint:errcheck
	baseLogger := logger.With(

		zap.String("service", "api-gateway"),
		zap.String("env", os.Getenv("ENV")),
		zap.String("version", "buildVersionXXX"),
	)

	// ---- gRPC
	tlsDialOpt, err := clientCredsToCRM()
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println("temporarily while comment passing it to gprc opts ", tlsDialOpt)

	conn, err := grpc.NewClient(
		"localhost:50051",
		tlsDialOpt,
		//grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO: TLS позже
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

	// ---- HTTP main router
	r := chi.NewRouter()
	r.Use(
		//middleware.RequestID,
		middleware.Recoverer,
		//middleware.Logger,
		middleware.Compress(5),
		httpmdw.WithRequestID,
		httpmdw.WithIdempotencyKey,
		httpmdw.WithZapLogger(baseLogger),
		httpmdw.WithMetricsChi(cols),
	)

	mainSrv := &http.Server{
		Addr:              httpAddr,
		Handler:           r,
		ReadTimeout:       readTO,
		ReadHeaderTimeout: readTO,
		WriteTimeout:      writeTO,
		IdleTimeout:       idleTO,
	}
	r.With(
		bind.WithDTO(gwdto.BindCRMSearchQuery), // query → DTO + Validate()
		// bind.WithDTO(BindHeaders), bind.WithDTO(BindCookies), bind.WithDTO(BindPath) — добавим по мере надобности
	).Get("/crm/certificates/search", crmhandlers.Search(certsUC))

	// ---- HTTP admin router
	admin := chi.NewRouter()
	mountAdminHTTP(admin, reg, conn)
	adminSrv := &http.Server{
		Addr:              adminAddr,
		Handler:           admin,
		ReadHeaderTimeout: 2 * time.Second,
	}
	//srv := httpapi.NewServer(r, reqLog, reg, cols, certsUC)

	// ---- run both servers
	errCh := make(chan error, 2)
	go func() {
		logger.Info("http_listen", zap.String("addr", httpAddr))
		if err := mainSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		logger.Info("admin_listen", zap.String("addr", adminAddr))
		if err := adminSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// ---- graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done(): // получили сигнал
		baseLogger.Info("shutdown_begin")
	case err := <-errCh: // один из серверов упал
		baseLogger.Error("server_exit", zap.Error(err))
	}

	shCtx, cancel := context.WithTimeout(context.Background(), shutdownTO)
	defer cancel()

	_ = mainSrv.Shutdown(shCtx)  // перестаёт принимать новые, ждёт активные
	_ = adminSrv.Shutdown(shCtx) // закрывает admin

	baseLogger.Info("shutdown_end")

}

// conn — *grpc.ClientConn к CRM, который ты уже создаёшь
func mountAdminHTTP(r *chi.Mux, reg *prometheus.Registry, conn *grpc.ClientConn) {
	// liveness
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// readiness: канал к CRM должен быть Ready
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		state := conn.GetState()
		// обычно ждём именно Ready; Idle/Connecting — считаем «ещё не готов»
		if state != connectivity.Ready {
			http.Error(w, "crm grpc not ready: "+state.String(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// метрики уже смонтированы у тебя, оставляем как есть
	r.Method("GET", "/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
}
