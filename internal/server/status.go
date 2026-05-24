package server

import "sync"

type Status struct {
	mu      sync.RWMutex
	healthy bool
	ready   bool
}

func (s *Status) SetHealthy(healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = healthy
}

func (s *Status) SetReady(ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = ready
}

func (s *Status) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy
}

func (s *Status) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}