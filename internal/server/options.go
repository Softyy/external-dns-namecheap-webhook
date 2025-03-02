package server

import (
	"fmt"
	"time"

	"github.com/codingconcepts/env"
)

// SocketOptions contains the argument passed as environment variables that
// influence the socket configuration.
type SocketOptions struct {
	// Webhook host
	WebhookHost string `env:"WEBHOOK_HOST" default:"localhost"`
	// Webhook port
	WebhookPort uint16 `env:"WEBHOOK_PORT" default:"8888"`
	// Readiness and liveness probe host
	MetricsHost string `env:"METRICS_HOST" default:"0.0.0.0"`
	// Readiness and liveness probe port
	MetricsPort uint16 `env:"METRICS_PORT" default:"8080"`
	// Read timeout in milliseconds
	ReadTimeout int `env:"READ_TIMEOUT" default:"60000"`
	// Write timeout in milliseconds
	WriteTimeout int `env:"WRITE_TIMEOUT" default:"60000"`
}

// NewSocketOptions returns a pointer to a new SocketOptions instance. This
// instance will be populated with values taken from the relevant environment
// variables and a warning will be issued if a deprecated environment variable
// is used. If there is an issue while reading the environment variables
// an error is raised.
func NewSocketOptions() (*SocketOptions, error) {
	opt := &SocketOptions{}

	// Populate with values from environment.
	if err := env.Set(opt); err != nil {
		return nil, err
	}

	return opt, nil
}

// GetWebhookAddress returns the webhook socket address.
func (o SocketOptions) GetWebhookAddress() string {
	return fmt.Sprintf("%s:%d", o.WebhookHost, o.WebhookPort)
}

// GetHealthAddress returns the metrics socket address.
func (o SocketOptions) GetMetricsAddress() string {
	return fmt.Sprintf("%s:%d", o.MetricsHost, o.MetricsPort)
}

// GetReadTimeout returns the read timeout in milliseconds.
func (o SocketOptions) GetReadTimeout() time.Duration {
	return time.Duration(o.ReadTimeout) * time.Millisecond
}

// GetWriteTimeout returns the read timeout in milliseconds.
func (o SocketOptions) GetWriteTimeout() time.Duration {
	return time.Duration(o.WriteTimeout) * time.Millisecond
}
