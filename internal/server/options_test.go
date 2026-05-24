package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSocketOptions(t *testing.T) {
	opt, err := NewSocketOptions()
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "localhost", opt.WebhookHost)
	assert.Equal(t, uint16(8888), opt.WebhookPort)
	assert.Equal(t, "0.0.0.0", opt.MetricsHost)
	assert.Equal(t, uint16(8080), opt.MetricsPort)
}

func TestSocketOptions_GetWebhookAddress(t *testing.T) {
	opt := SocketOptions{
		WebhookHost: "0.0.0.0",
		WebhookPort: 9999,
	}
	assert.Equal(t, "0.0.0.0:9999", opt.GetWebhookAddress())
}

func TestSocketOptions_GetMetricsAddress(t *testing.T) {
	opt := SocketOptions{
		MetricsHost: "0.0.0.0",
		MetricsPort: 8080,
	}
	assert.Equal(t, "0.0.0.0:8080", opt.GetMetricsAddress())
}

func TestSocketOptions_Timeouts(t *testing.T) {
	opt := SocketOptions{
		ReadTimeout:  30000,
		WriteTimeout: 45000,
	}
	assert.Equal(t, 30000, int(opt.GetReadTimeout().Milliseconds()))
	assert.Equal(t, 45000, int(opt.GetWriteTimeout().Milliseconds()))
}