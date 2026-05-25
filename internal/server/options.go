package server

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type SocketOptions struct {
	WebhookHost  string `env:"WEBHOOK_HOST" envDefault:"localhost"`
	WebhookPort  uint16 `env:"WEBHOOK_PORT" envDefault:"8888"`
	MetricsHost  string `env:"METRICS_HOST" envDefault:"0.0.0.0"`
	MetricsPort  uint16 `env:"METRICS_PORT" envDefault:"8080"`
	ReadTimeout  int    `env:"READ_TIMEOUT" envDefault:"60000"`
	WriteTimeout int    `env:"WRITE_TIMEOUT" envDefault:"60000"`
}

func NewSocketOptions() (*SocketOptions, error) {
	opt := &SocketOptions{}
	if err := env.Parse(opt); err != nil {
		return nil, err
	}
	return opt, nil
}

func (o SocketOptions) GetWebhookAddress() string {
	return fmt.Sprintf("%s:%d", o.WebhookHost, o.WebhookPort)
}

func (o SocketOptions) GetMetricsAddress() string {
	return fmt.Sprintf("%s:%d", o.MetricsHost, o.MetricsPort)
}

func (o SocketOptions) GetReadTimeout() time.Duration {
	return time.Duration(o.ReadTimeout) * time.Millisecond
}

func (o SocketOptions) GetWriteTimeout() time.Duration {
	return time.Duration(o.WriteTimeout) * time.Millisecond
}