package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type CRM struct {
	SearchTotal    *prometheus.CounterVec   // result: ok|error
	SearchDuration *prometheus.HistogramVec // секунды
}

func Register(reg *prometheus.Registry) *CRM {
	m := &CRM{
		SearchTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "crm_search_total",
				Help: "Total number of CRM search operations by result.",
			},
			[]string{"result"},
		),
		SearchDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "crm_search_duration_seconds",
				Help:    "Duration of CRM search operations.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"result"},
		),
	}
	reg.MustRegister(m.SearchTotal, m.SearchDuration)
	return m
}
