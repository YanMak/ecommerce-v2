package cbreaker

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrCircuitOpen = errors.New("circuit breaker open")

type Config struct {
	Enabled              bool
	Target               string        // метка (например, "crm")
	Interval             time.Duration // окно статистики (сброс счётчиков)
	Timeout              time.Duration // сколько держать OPEN
	HalfOpenMaxRequests  uint32        // сколько «пробных» в HALF-OPEN
	MinSamples           uint32        // минимум запросов, чтобы включить правило error-rate
	ErrorRate            float64       // доля ошибок для трипа (0.6 = 60%)
	MaxConsecutiveErrors uint32        // подряд ошибок для трипа
}

type metrics struct {
	state   *prometheus.GaugeVec   // 0=closed,1=half_open,2=open
	trips   *prometheus.CounterVec // переходы в OPEN
	rejects *prometheus.CounterVec // отказов из-за OPEN/HALF-OPEN
	target  string
}

func newMetrics(reg prometheus.Registerer, target string) *metrics {
	m := &metrics{
		state: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_cb_state",
				Help: "Circuit breaker state (0=closed,1=half_open,2=open).",
			},
			[]string{"target"},
		),
		trips: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_cb_trips_total",
				Help: "Number of circuit breaker trips.",
			},
			[]string{"target"},
		),
		rejects: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_cb_rejected_total",
				Help: "Number of requests rejected by an open circuit.",
			},
			[]string{"target"},
		),
		target: target,
	}
	if reg != nil {
		reg.MustRegister(m.state, m.trips, m.rejects)
	}
	// начальное состояние = closed
	m.state.WithLabelValues(target).Set(0)
	return m
}

type Breaker struct {
	cb  *gobreaker.CircuitBreaker
	m   *metrics
	cfg Config
}

func New(reg prometheus.Registerer, cfg Config) *Breaker {
	m := newMetrics(reg, cfg.Target)

	readyToTrip := func(c gobreaker.Counts) bool {
		// 1) подряд ошибок
		if cfg.MaxConsecutiveErrors > 0 && c.ConsecutiveFailures >= cfg.MaxConsecutiveErrors {
			return true
		}
		// 2) доля ошибок при достаточном числе запросов
		if cfg.MinSamples > 0 && c.Requests >= cfg.MinSamples {
			totalErrs := c.TotalFailures
			errRate := float64(totalErrs) / float64(c.Requests)
			if errRate >= cfg.ErrorRate {
				return true
			}
		}
		return false
	}

	st2num := func(s gobreaker.State) float64 {
		switch s {
		case gobreaker.StateClosed:
			return 0
		case gobreaker.StateHalfOpen:
			return 1
		case gobreaker.StateOpen:
			return 2
		default:
			return -1
		}
	}

	settings := gobreaker.Settings{
		Name:        cfg.Target,
		MaxRequests: cfg.HalfOpenMaxRequests, // в HALF-OPEN
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: readyToTrip,
		OnStateChange: func(_ string, from, to gobreaker.State) {
			m.state.WithLabelValues(cfg.Target).Set(st2num(to))
			// считаем "trip" как переход в OPEN
			if to == gobreaker.StateOpen {
				m.trips.WithLabelValues(cfg.Target).Inc()
			}
		},
	}

	return &Breaker{
		cb:  gobreaker.NewCircuitBreaker(settings),
		m:   m,
		cfg: cfg,
	}
}

// Execute оборачивает fn.
// - Ошибки, подходящие под "срыв" (retryable/транзиентные), считаются ошибками для CB.
// - Бизнес-ошибки (не retryable) НЕ учитываются как ошибки для CB (хоть и возвращаются вызывающему).
func (b *Breaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if !b.cfg.Enabled {
		return fn(ctx)
	}

	val, err := b.cb.Execute(func() (interface{}, error) {
		e := fn(ctx)
		if e == nil {
			return nil, nil
		}
		// Если ошибка НЕ триггерная — не считаем её ошибкой для CB.
		if !shouldTrip(e) {
			// Вернём ошибку как значение, но error=nil, чтобы CB посчитал «успех».
			return e, nil
		}
		// Триггерная ошибка — пусть CB считает fail.
		return nil, e
	})

	// OPEN/HALF-OPEN отфутболили нас
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			b.m.rejects.WithLabelValues(b.cfg.Target).Inc()
			return ErrCircuitOpen
		}
		return err
	}

	// CB считает успех — но мы могли вернуть бизнес-ошибку как значение
	if val != nil {
		if e, ok := val.(error); ok && e != nil {
			return e
		}
	}
	return nil
}

// shouldTrip — классификация «плохих» ошибок, которые должны ломать цепь.
func shouldTrip(err error) bool {
	// контекст
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	// gRPC коды
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Internal:
			return true
		default:
			return false
		}
	}
	return false
}
