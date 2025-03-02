package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/external-dns/provider/webhook/api"

	"external-dns/webhooks/namecheap/internal/namecheap"
	"external-dns/webhooks/namecheap/internal/server"
)

var (
	// Version compiled by goreleaser.
	Version = "dev"
	// Gitsha (commit SHA1) compiled by goreleaser.
	Gitsha = "none"
)

// notify requires the SIGINT and SIGTERM signals to be sent to the caller.
var notify = func(sig chan os.Signal) {
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
}

// healthStatus is the interface used by loop.
type healthStatus interface {
	SetHealthy(bool)
	SetReady(bool)
}

// waitForSignal waits for a SIGTERM or a SIGINT and then shuts down the server.
func waitForSignal(status healthStatus) {
	exitSignal := make(chan os.Signal, 1)
	notify(exitSignal)
	signal := <-exitSignal

	log.Infof("Signal %s received. Shutting down the webhook.", signal.String())
	status.SetHealthy(false)
	status.SetReady(false)
}

func main() {
	log.Infof("Starting Hetzner webhook version %s (commit %s)", Version, Gitsha)
	// Read server options
	socketOptions, err := server.NewSocketOptions()
	if err != nil {
		log.Fatal("Cannot read configuration from environment:", err.Error())
		log.Exit(1)
	}

	// Start health server
	log.Infof("Starting metrics server with socket address %s", socketOptions.GetMetricsAddress())
	serverStatus := server.Status{}
	serverStatus.SetHealthy(true)
	metricsSocket := server.NewMetricsSocket(&serverStatus)
	go metricsSocket.Start(nil, *socketOptions)

	// Read provider configuration

	}

	// instantiate the Namecheap provider
	provider := NewNamecheapProvider(config)

	// Start the webhook
	log.Infof("Starting webhook server with socket address %s", socketOptions.GetWebhookAddress())
	startedChan := make(chan struct{})
	go api.StartHTTPApi(
		provider, startedChan,
		socketOptions.GetReadTimeout(),
		socketOptions.GetWriteTimeout(),
		socketOptions.GetWebhookAddress(),
	)

	// Wait for the HTTP server to start and then set the healthy and ready flags
	<-startedChan
	serverStatus.SetReady(true)

	// Wait until a signal tells us to exit
	waitForSignal(&serverStatus)
}
