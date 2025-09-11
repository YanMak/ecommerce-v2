package grpcx

import (
	"context"
	"strings"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// /package.Service/Method -> ("package.Service", "Method")
func splitFullMethod(full string) (string, string) {
	// full: "/pkg.svc.Service/Method"
	if len(full) == 0 {
		return "", ""
	}
	full = strings.TrimPrefix(full, "/")
	parts := strings.SplitN(full, "/", 2)
	svc, m := parts[0], ""
	if len(parts) > 1 {
		m = parts[1]
	}
	return svc, m
}

func UnaryClientMetricsInterceptor(c *prom.Collectors) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		svc, m := splitFullMethod(method)
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		sec := time.Since(start).Seconds()

		st, _ := status.FromError(err)
		code := st.Code().String()

		c.GRPCClientTotal.WithLabelValues(svc, m, code).Inc()
		c.GRPCClientDuration.WithLabelValues(svc, m).Observe(sec)
		return err
	}
}

// UnaryServerMetricsInterceptor — считает серверные вызовы.
func UnaryServerMetricsInterceptor(c *prom.Collectors) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		svc, m := splitFullMethod(info.FullMethod)
		start := time.Now()
		resp, err := handler(ctx, req)
		sec := time.Since(start).Seconds()

		st, _ := status.FromError(err)
		code := st.Code().String()
		if code == "" {
			code = codes.OK.String()
		}

		// Можно завести отдельные векторы для server-*; для краткости используем client-метрики или добавь в Collectors ещё два вектора:
		c.GRPCServerTotal.WithLabelValues(svc, m, code).Inc()
		c.GRPCServerDuration.WithLabelValues(svc, m).Observe(sec)
		return resp, err
	}
}
