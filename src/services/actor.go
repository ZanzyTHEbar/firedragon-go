// Package services provides service implementations using the hollywood actor model
package services

import (
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/anthdm/hollywood/actor"
)

// ActorService is the interface that all actor-based services implement
type ActorService interface {
	actor.Receiver
}

// BaseActor provides common functionality for all actors
type BaseActor struct {
	logger *internal.Logger
	name   string
}

// NewBaseActor creates a new base actor with the given name and logger
func NewBaseActor(name string, logger *internal.Logger) BaseActor {
	return BaseActor{
		name:   name,
		logger: logger,
	}
}

// StartMsg tells an actor to start processing
type StartMsg struct{}

// StopMsg tells an actor to stop processing
type StopMsg struct{}

// StatusRequestMsg is a message requesting the current status of an actor
type StatusRequestMsg struct{}

// StatusResponseMsg is the response to a status request
type StatusResponseMsg struct {
	Status      interfaces.ServiceStatus
	LastActive  time.Time
	ErrorCount  int
	LastError   error
	CustomStats map[string]interface{}
}
