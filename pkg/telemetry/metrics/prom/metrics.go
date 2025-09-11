package prom

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Collectors struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	GRPCClientTotal     *prometheus.CounterVec
	GRPCClientDuration  *prometheus.HistogramVec
	GRPCServerTotal     *prometheus.CounterVec
	GRPCServerDuration  *prometheus.HistogramVec
}

// New создает реестр и регистрирует коллекции.
func New() (*prometheus.Registry, *Collectors) {
	reg := prometheus.NewRegistry()

	c := &Collectors{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"route", "method", "code"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"route", "method"},
		),
		GRPCClientTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_client_requests_total",
				Help: "Total number of gRPC client calls.",
			},
			[]string{"service", "method", "code"},
		),
		GRPCClientDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "grpc_client_duration_seconds",
				Help:    "Duration of gRPC client calls.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method"},
		),
		GRPCServerTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_server_requests_total",
				Help: "Total number of gRPC server calls.",
			},
			[]string{"service", "method", "code"},
		),
		GRPCServerDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "grpc_server_duration_seconds",
				Help:    "Duration of gRPC server calls.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method"},
		),
	}

	reg.MustRegister(
		c.HTTPRequestsTotal,
		c.HTTPRequestDuration,
		c.GRPCClientTotal,
		c.GRPCClientDuration,
		c.GRPCServerTotal,
		c.GRPCServerDuration,
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)

	return reg, c
}
