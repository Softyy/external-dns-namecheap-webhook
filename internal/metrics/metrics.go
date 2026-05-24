package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var metrics *OpenMetrics

type OpenMetrics struct {
	registry *prometheus.Registry
	handler  http.Handler

	successfulApiCallsTotal *prometheus.CounterVec
	failedApiCallsTotal     *prometheus.CounterVec

	filteredOutZones prometheus.Gauge
	skippedRecords   *prometheus.GaugeVec
	apiDelayHist     *prometheus.HistogramVec
}

func GetOpenMetricsInstance() *OpenMetrics {
	if metrics == nil {
		reg := prometheus.NewRegistry()
		metrics = &OpenMetrics{
			registry: reg,
			handler:  promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
			successfulApiCallsTotal: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "namecheap_successful_api_calls_total",
					Help: "The number of successful Namecheap API calls",
				},
				[]string{"action"},
			),
			failedApiCallsTotal: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "namecheap_failed_api_calls_total",
					Help: "The number of Namecheap API calls that returned an error",
				},
				[]string{"action"},
			),
			filteredOutZones: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "namecheap_filtered_out_zones",
				Help: "The number of zones excluded by the domain filter",
			}),
			skippedRecords: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "namecheap_skipped_records",
					Help: "The number of skipped records per domain",
				},
				[]string{"zone"},
			),
			apiDelayHist: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "namecheap_api_delay_hist",
					Help:    "Histogram of the delay in milliseconds when calling the Namecheap API",
					Buckets: []float64{10, 100, 250, 500, 1000, 1500, 2000},
				},
				[]string{"action"},
			),
		}
		reg.MustRegister(metrics.successfulApiCallsTotal)
		reg.MustRegister(metrics.failedApiCallsTotal)
		reg.MustRegister(metrics.filteredOutZones)
		reg.MustRegister(metrics.skippedRecords)
		reg.MustRegister(metrics.apiDelayHist)
	}
	return metrics
}

func (m *OpenMetrics) GetHandler() http.Handler {
	return m.handler
}

func (m *OpenMetrics) GetRegistry() *prometheus.Registry {
	return m.registry
}

func (m *OpenMetrics) IncSuccessfulApiCallsTotal(action string) {
	label := prometheus.Labels{"action": action}
	m.successfulApiCallsTotal.With(label).Inc()
}

func (m *OpenMetrics) IncFailedApiCallsTotal(action string) {
	label := prometheus.Labels{"action": action}
	m.failedApiCallsTotal.With(label).Inc()
}

func (m *OpenMetrics) SetFilteredOutZones(num int) {
	m.filteredOutZones.Set(float64(num))
}

func (m *OpenMetrics) SetSkippedRecords(zone string, num int) {
	label := prometheus.Labels{"zone": zone}
	m.skippedRecords.With(label).Set(float64(num))
}

func (m *OpenMetrics) AddApiDelayHist(action string, delay int64) {
	label := prometheus.Labels{"action": action}
	m.apiDelayHist.With(label).Observe(float64(delay))
}