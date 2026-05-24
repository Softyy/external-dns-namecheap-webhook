package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatus_SetHealthy(t *testing.T) {
	s := Status{}
	s.SetHealthy(true)
	assert.True(t, s.IsHealthy())

	s.SetHealthy(false)
	assert.False(t, s.IsHealthy())
}

func TestStatus_SetReady(t *testing.T) {
	s := Status{}
	s.SetReady(true)
	assert.True(t, s.IsReady())

	s.SetReady(false)
	assert.False(t, s.IsReady())
}

func TestStatus_Concurrent(t *testing.T) {
	s := Status{}

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			s.SetHealthy(true)
			s.SetHealthy(false)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = s.IsHealthy()
			_ = s.IsReady()
		}
		done <- true
	}()

	<-done
	<-done
}