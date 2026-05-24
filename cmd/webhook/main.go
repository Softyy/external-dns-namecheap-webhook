package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

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
	signal := <-exitSignal

	log.Infof("Signal %s received. Shutting down the webhook.", signal.String())
	status.SetHealthy(false)
	status.SetReady(false)
}

func main() {
	log.Infof("Starting Namecheap webhook version %s (commit %s)", Version, Gitsha)

	socketOptions, err := server.NewSocketOptions()
	if err != nil {
		log.Fatalf("Cannot read configuration from environment: %s", err.Error())
	}

	log.Infof("Starting metrics server with socket address %s", socketOptions.GetMetricsAddress())
	serverStatus := server.Status{}
	serverStatus.SetHealthy(true)
	metricsSocket := server.NewMetricsSocket(&serverStatus)
	go metricsSocket.Start(nil, *socketOptions)

	config, err := namecheap.NewConfiguration()
	if err != nil {
		serverStatus.SetHealthy(false)
		log.Fatalf("Cannot read provider configuration: %s", err.Error())
	}

	provider, err := namecheap.NewNamecheapProvider(config)
	if err != nil {
		serverStatus.SetHealthy(false)
		log.Fatalf("Cannot create Namecheap provider: %s", err.Error())
	}

	log.Infof("Starting webhook server with socket address %s", socketOptions.GetWebhookAddress())
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