package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "evofarm_http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	jobsSubmittedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "evofarm_jobs_submitted_total",
			Help: "Total jobs submitted",
		},
	)

	authRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "evofarm_auth_requests_total",
			Help: "Total auth requests by type and status",
		},
		[]string{"type", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(jobsSubmittedTotal)
	prometheus.MustRegister(authRequestsTotal)
}

// metricsHandler returns the Prometheus scrape endpoint.
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}