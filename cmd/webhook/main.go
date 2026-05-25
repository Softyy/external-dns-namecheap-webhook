package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"sigs.k8s.io/external-dns/provider/webhook/api"

	"external-dns/webhooks/namecheap/internal/namecheap"
	"external-dns/webhooks/namecheap/internal/server"
)

var (
	Version = "dev"
	Gitsha  = "none"
)

var notify = func(sig chan os.Signal) {
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
}

type healthStatus interface {
	SetHealthy(bool)
	SetReady(bool)
}

func waitForSignal(status healthStatus) {
	exitSignal := make(chan os.Signal, 1)
	notify(exitSignal)
	sig := <-exitSignal

	slog.Info("Signal received, shutting down the webhook", "signal", sig.String())
	status.SetHealthy(false)
	status.SetReady(false)
}

func main() {
	slog.Info("Starting Namecheap webhook", "version", Version, "commit", Gitsha)

	socketOptions, err := server.NewSocketOptions()
	if err != nil {
		slog.Error("Cannot read configuration from environment", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting metrics server", "address", socketOptions.GetMetricsAddress())
	serverStatus := server.Status{}
	serverStatus.SetHealthy(true)
	metricsSocket := server.NewMetricsSocket(&serverStatus)
	go metricsSocket.Start(nil, *socketOptions)

	config, err := namecheap.NewConfiguration()
	if err != nil {
		serverStatus.SetHealthy(false)
		slog.Error("Cannot read provider configuration", "error", err)
		os.Exit(1)
	}

	if config.Debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

	provider, err := namecheap.NewNamecheapProvider(config)
	if err != nil {
		serverStatus.SetHealthy(false)
		slog.Error("Cannot create Namecheap provider", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting webhook server", "address", socketOptions.GetWebhookAddress())
	startedChan := make(chan struct{})
	go api.StartHTTPApi(
		provider, startedChan,
		socketOptions.GetReadTimeout(),
		socketOptions.GetWriteTimeout(),
		socketOptions.GetWebhookAddress(),
	)

	<-startedChan
	serverStatus.SetReady(true)

	waitForSignal(&serverStatus)
}