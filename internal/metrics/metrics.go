package metrics

import (
	"regexp"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// RequestDuration tracks HTTP request duration in seconds by method, path, status.
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// RequestTotal counts HTTP requests by method, path, status.
	RequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// ScanJobsRunning is the number of scans currently running (in-memory).
	ScanJobsRunning = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "scan_jobs_running",
			Help: "Number of scan jobs currently running",
		},
	)

	// ScanJobsTotal counts scan job completions by status (completed, canceled, error).
	ScanJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scan_jobs_total",
			Help: "Total number of scan jobs finished by status",
		},
		[]string{"status"},
	)
)

var (
	numericPathSegment = regexp.MustCompile(`/[0-9]+(/|$)`)
	initOnce           sync.Once
)

func init() {
	initOnce.Do(func() {
		prometheus.MustRegister(RequestDuration, RequestTotal, ScanJobsRunning, ScanJobsTotal)
	})
}

// NormalizePath reduces cardinality by replacing numeric path segments with {id}.
// E.g. /v1/assets/123 -> /v1/assets/{id}, /v1/scans/45 -> /v1/scans/{id}.
func NormalizePath(path string) string {
	return numericPathSegment.ReplaceAllString(path, "/{id}$1")
}

// RecordRequest records duration and count for an HTTP request. Call from middleware with method, path, statusCode, duration.
func RecordRequest(method, path string, statusCode int, durationSeconds float64) {
	path = NormalizePath(path)
	status := strconv.Itoa(statusCode)
	RequestDuration.WithLabelValues(method, path, status).Observe(durationSeconds)
	RequestTotal.WithLabelValues(method, path, status).Inc()
}

// IncScanJobsRunning increments the running scan jobs gauge (call when a scan starts).
func IncScanJobsRunning() {
	ScanJobsRunning.Inc()
}

// DecScanJobsRunning decrements the running scan jobs gauge (call when a scan finishes).
func DecScanJobsRunning() {
	ScanJobsRunning.Dec()
}

// IncScanJobsTotal increments the scan jobs counter for the given status (completed, canceled, error).
func IncScanJobsTotal(status string) {
	ScanJobsTotal.WithLabelValues(status).Inc()
}
