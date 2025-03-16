// Package services provides service implementations using the hollywood actor model
package services

import (
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/anthdm/hollywood/actor"
)

// ActorService is the interface that all actor-based services implement
type ActorService interface {
	actor.Receiver
}

// Message types for actor communication
type StartMsg struct{}
type StopMsg struct{}
type StatusRequestMsg struct{}

type StatusResponseMsg struct {
	Status      string
	LastActive  time.Time
	ErrorCount  int
	LastError   error
	CustomStats map[string]interface{}
}

// BaseActor provides common functionality for all actors
type BaseActor struct {
	logger     *internal.Logger
	name       string
	status     string
	startTime  time.Time
	errorCount int
	lastError  error
	stats      map[string]interface{}
}

// NewBaseActor creates a new base actor with initialized fields
func NewBaseActor(name string, logger *internal.Logger) *BaseActor {
	return &BaseActor{
		name:       name,
		logger:     logger,
		status:     "initialized",
		startTime:  time.Now(),
		stats:      make(map[string]interface{}),
	}
}

// HandleStatus processes status request messages
func (a *BaseActor) HandleStatus() *StatusResponseMsg {
	return &StatusResponseMsg{
		Status:      a.status,
		LastActive:  a.startTime,
		ErrorCount:  a.errorCount,
		LastError:   a.lastError,
		CustomStats: a.stats,
	}
}

// UpdateStatus sets the current actor status
func (a *BaseActor) UpdateStatus(status string) {
	a.status = status
}

// RecordError increments error count and stores last error
func (a *BaseActor) RecordError(err error) {
	a.errorCount++
	a.lastError = err
}

// UpdateStats adds or updates custom statistics
func (a *BaseActor) UpdateStats(key string, value interface{}) {
	a.stats[key] = value
}
