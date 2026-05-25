package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"external-dns/webhooks/namecheap/internal/metrics"
)

type HealthStatus interface {
	IsHealthy() bool
	IsReady() bool
}

type MetricsSocket struct {
	status HealthStatus
}

func NewMetricsSocket(status HealthStatus) *MetricsSocket {
	return &MetricsSocket{
		status: status,
	}
}

func (s *MetricsSocket) Start(ctx context.Context, options SocketOptions) {
	addr := fmt.Sprintf("%s:%d", options.MetricsHost, options.MetricsPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if s.status.IsHealthy() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Unhealthy")
		}
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if s.status.IsReady() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Not Ready")
		}
	})

	metricsHandler := metrics.GetOpenMetricsInstance().GetHandler()
	if metricsHandler != nil {
		mux.Handle("/metrics", metricsHandler)
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		slog.Info("Starting metrics server", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start metrics server", "error", err)
		}
	}()

	if ctx != nil {
		go func() {
			<-ctx.Done()
			slog.Info("Shutting down metrics server")
			if err := srv.Shutdown(context.Background()); err != nil {
				slog.Error("Failed to shutdown metrics server", "error", err)
			}
		}()
	}
}