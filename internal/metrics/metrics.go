package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SIPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sipplane_sip_requests_total",
		Help: "SIP requests handled by method, tenant, route, trunk, and response code",
	}, []string{"method", "tenant", "route", "trunk", "code"})

	SIPTransactionsInflight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sipplane_sip_transactions_inflight",
		Help: "In-flight SIP server transactions",
	}, []string{"method"})

	RegisterBindings = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sipplane_register_bindings",
		Help: "Active REGISTER bindings",
	}, []string{"tenant"})

	ConfigRevision = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sipplane_config_revision",
		Help: "Current applied configuration revision",
	})

	ConfigApplyTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sipplane_config_apply_total",
		Help: "Config apply attempts by result",
	}, []string{"result"})

	LocationLookups = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sipplane_location_lookup_total",
		Help: "Location lookups by result",
	}, []string{"result"})

	RateLimitRejected = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sipplane_rate_limit_rejected_total",
		Help: "Ingress requests rejected by rate limit",
	}, []string{"backend", "key"})
)

// ObserveRequest increments request counters.
func ObserveRequest(method, tenant, route, trunk string, code int) {
	SIPRequestsTotal.WithLabelValues(method, label(tenant), label(route), label(trunk), itoa(code)).Inc()
}

func label(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
