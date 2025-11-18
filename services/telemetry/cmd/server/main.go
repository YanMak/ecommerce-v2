package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	"github.com/YanMak/ecommerce/v2/pkg/otelx"
	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"
	grpcin "github.com/YanMak/ecommerce/v2/services/telemetry/internal/adapters/inbound/grpc"
	"github.com/YanMak/ecommerce/v2/services/telemetry/internal/app/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	grpcx "github.com/YanMak/ecommerce/v2/pkg/grpcx"

	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"

	crmmetrics "github.com/YanMak/ecommerce/v2/services/telemetry/internal/metrics"

	cfg "github.com/YanMak/ecommerce/v2/pkg/config"

	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	otelpgx "github.com/exaring/otelpgx"

	httpmdw "github.com/YanMak/ecommerce/v2/pkg/httpx/middleware"
)

func serverTLSCreds() (grpc.ServerOption, error) {
	certFile := getenv("TLS_CERT_FILE", "/etc/enterprise/tls/crm/crm.pem")
	keyFile := getenv("TLS_KEY_FILE", "/etc/enterprise/tls/crm/crm.key")
	clientCA := getenv("TLS_CLIENT_CA_FILE", "/etc/enterprise/tls/ca/ca.pem")

	// Пул доверенных CA для проверки клиентских сертификатов
	caPEM, err := os.ReadFile(clientCA)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("append CA failed")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		// ALPN h2 gRPC добавит сам через credentials.NewTLS
		ClientCAs:  pool,
		ClientAuth: tls.RequireAndVerifyClientCert, // ← включаем mTL
	}
	return grpc.Creds(credentials.NewTLS(tlsCfg)), nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// func runGRPC(addr string, pool *pgxpool.Pool, cols *prom.Collectors, logger *zap.Logger, crmM *crmmetrics.CRM) error {
// 	lis, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		return err
// 	}

// 	s := grpc.NewServer(
// 		grpc.ChainUnaryInterceptor(
// 			grpcx.UnaryServerMetaInterceptor,
// 			// тут позже можно добавить лог/метрики/рековери-интерсепторы
// 			grpcx.UnaryServerZapLogger(logger),
// 			grpcx.UnaryServerMetricsInterceptor(cols),
// 		),
// 		grpc.ChainStreamInterceptor(
// 			grpcx.StreamServerMetaInterceptor, // если будут streaming RPC
// 		),
// 	)
// 	uc := usecase.NewCertificatesUC(pool, crmM)
// 	crmpb.RegisterCertificatesServer(s, grpcin.NewCertificatesServer(uc))

// 	return s.Serve(lis)
// }

func main() {
	grpcAddr := cfg.Str("GRPC_ADDR", ":50051")
	adminAddr := cfg.Str("ADMIN_ADDR", ":8089")
	shutdownTO := cfg.Dur("SHUTDOWN_TIMEOUT", 10*time.Second)
	drainDelay := cfg.Dur("DRAIN_DELAY", 2*time.Second) // короткая пауза на дренаж
	drainTO := cfg.Dur("DRAIN_TIMEOUT", 8*time.Second)  // максимум на дренаж gRPC

	// ---- Pgx
	dsn := "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable"
	ctx := context.Background()

	//v1
	// // Разбираем конфиг пула и включаем OTel-трейсер для pgx (db-спаны):
	// cfg, err := pgxpool.ParseConfig(dsn)
	// if err != nil {
	// 	log.Fatal("pgxpool ParseConfig:", err)
	// }
	// cfg.ConnConfig.Tracer = otelpgx.NewTracer(
	// 	// В проде лучше не класть полный SQL в атрибуты:
	// 	otelpgx.WithDisableSQLStatementInAttributes(),
	// 	// Если нужно в деве — можно временно показать параметры:
	// 	// otelpgx.WithIncludeQueryParameters(),
	// )
	// pool, err := pgxpool.NewWithConfig(ctx, cfg)
	// v2
	// ---- Pgx
	cfgPg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("pgxpool ParseConfig:", err)
	}
	// Трейсинг запросов (db-спаны)
	cfgPg.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithDisableSQLStatementInAttributes(), // прод-безопасно
	)
	// Жёсткий предел на длительность SQL внутри PG (дополнительно к ctx):
	// читаем TTL из ENV: DB_STATEMENT_TIMEOUT (напр. "2s").
	stmtTO := cfg.Dur("DB_STATEMENT_TIMEOUT", 20*time.Second)
	if stmtTO > 0 {
		cfgPg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {

			// env examples
			// DB_STATEMENT_TIMEOUT=2s
			// DB_LOCK_TIMEOUT=500ms
			// DB_IDLE_IN_TX_TIMEOUT=15s

			// PG принимает 'Nms' / 'Ns' как literal
			_, err := c.Exec(ctx, fmt.Sprintf("SET statement_timeout = '%dms'", stmtTO.Milliseconds()))

			// (опционально) дополнительные предохранители:
			if lockTO := cfg.Dur("DB_LOCK_TIMEOUT", 0); lockTO > 0 {
				if _, err := c.Exec(ctx, fmt.Sprintf("SET lock_timeout = '%dms'", lockTO.Milliseconds())); err != nil {
					return err
				}
			}
			if idleTxTO := cfg.Dur("DB_IDLE_IN_TX_TIMEOUT", 0); idleTxTO > 0 {
				if _, err := c.Exec(ctx, fmt.Sprintf("SET idle_in_transaction_session_timeout = '%dms'", idleTxTO.Milliseconds())); err != nil {
					return err
				}
			}

			return err
		}
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfgPg)
	if err != nil {
		log.Fatal("pgxpool NewWithConfig:", err)
	}

	defer pool.Close()

	// ---- telemetry
	reg, cols := prom.New()
	crmM := crmmetrics.Register(reg)
	//base, _ := tlog.NewProduction()
	base, err := tlog.NewLogger(cfg.Str("DEV_LOG_FILE", "dev-logs/telemetry-service-simple-01.log"))
	base = base.With(
		zap.String("service", "telemetry"),
		zap.String("env", os.Getenv("ENV")),
	)
	defer base.Sync()
	// OTEL {
	shutdown, err := otelx.InitTracer(ctx, "telemetry-grpc")
	if err != nil {
		panic(err)
	}
	defer shutdown(context.Background())
	// }

	// ---- usecases
	uc := usecase.NewCertificatesUC(pool, crmM)

	// ---- readiness flag
	var ready atomic.Bool
	ready.Store(true)

	// Yan 16 11 25 we disable tls due to inexistance of certs
	// // ---- TLS options
	// srvTLSOpt, err := serverTLSCreds()
	// if err != nil {
	// 	panic(err)
	// }
	//fmt.Println("temporarily while comment passing it to gprc opts ", srvTLSOpt)

	// ---- gRPC health
	hs := health.NewServer()

	// ---- gRPC server + interceptors (meta → ctx, request-logger, metrics)
	s := grpc.NewServer(
		// Yan we disable tls comminication due to unexistanse of tls certs
		//srvTLSOpt,
		grpc.StatsHandler(otelgrpc.NewServerHandler()), // ← создаёт серверные спаны для всех RPC
		grpc.ChainUnaryInterceptor(
			grpcx.UnaryServerMetaInterceptor,
			// тут позже можно добавить лог/метрики/рековери-интерсепторы
			grpcx.UnaryServerZapLogger(base),
			grpcx.UnaryServerMetricsInterceptor(cols),
		),
		grpc.ChainStreamInterceptor(
			grpcx.StreamServerMetaInterceptor, // если будут streaming RPC
		),
	)
	crmpb.RegisterCertificatesServer(s, grpcin.NewCertificatesServer(uc))
	grpc_health_v1.RegisterHealthServer(s, hs)
	// отмечаем как готовые (общий и целевой сервис)
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	hs.SetServingStatus(crmpb.Certificates_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)

	////////////////
	// HTTP admin
	admin := chi.NewRouter()
	admin.Use(
		middleware.Recoverer,
		middleware.Compress(5),
		httpmdw.WithRequestID,
		httpmdw.WithZapLogger(base),
	)
	//bindAdminHTTP(admin, pool, reg)
	// /livez — процесс жив
	admin.Get("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// /readyz — готов принимать трафик
	admin.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		// опционально: быстрый ping БД
		c, cancel := context.WithTimeout(r.Context(), 300*time.Millisecond)
		defer cancel()
		if err := pool.Ping(c); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// уже существующий экспорт метрик
	admin.Method("GET", "/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	adminSrv := &http.Server{Addr: adminAddr, Handler: admin}
	/////////////

	// run gRPC
	errCh := make(chan error, 2)
	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			errCh <- err
			return
		}
		base.Info("grpc_listen", zap.String("addr", grpcAddr))
		if err := s.Serve(lis); err != nil {
			errCh <- err
		}
	}()
	// run admin
	go func() {
		base.Info("admin_listen", zap.String("addr", adminAddr))
		if err := adminSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// graceful
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		base.Info("shutdown_begin")
	case err := <-errCh:
		base.Error("server_exit", zap.Error(err))
	}

	// 1) снимаем готовность и health — как у тебя было
	ready.Store(false)
	hs.Shutdown()
	time.Sleep(drainDelay) // короткая пауза, чтобы LB/ingress выпилили инстанс

	// 2) запускаем graceful в горутине и ждём с таймаутом
	done := make(chan struct{})
	go func() {
		s.GracefulStop() // перестаёт принимать новые RPC, ждёт активные
		close(done)
	}()

	select {
	case <-done:
		base.Info("grpc_graceful_stop_done")
	case <-time.After(drainTO):
		base.Warn("grpc_graceful_timeout_force_stop", zap.Duration("drain_timeout", drainTO))
		s.Stop() // форс-закрытие: активные RPC обрываются с UNAVAILABLE
	}

	// 3) закрываем admin http с дедлайном (можно тем же shCtx)
	shCtx, cancel := context.WithTimeout(context.Background(), shutdownTO)
	defer cancel()
	_ = adminSrv.Shutdown(shCtx)

	// 4) ресурсы
	pool.Close()
	base.Info("shutdown_end")

}

func bindAdminHTTP(r *chi.Mux, pool *pgxpool.Pool, reg *prometheus.Registry) {

	// liveness: просто жив
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// readiness: быстрый ping БД
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "db not ready: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// уже существующий экспорт метрик
	r.Method("GET", "/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
}
