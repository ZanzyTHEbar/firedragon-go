// Package services provides service implementations using the hollywood actor model
package services

import (
	"context"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/anthdm/hollywood/actor"
)

// ServiceStatus represents the current status of a service
type ServiceStatus string

const (
	// ServiceStatusRunning indicates the service is currently running
	ServiceStatusRunning ServiceStatus = "RUNNING"
	// ServiceStatusStopped indicates the service is currently stopped
	ServiceStatusStopped ServiceStatus = "STOPPED"
	// ServiceStatusError indicates the service has encountered an error
	ServiceStatusError ServiceStatus = "ERROR"
	// ServiceStatusUnknown indicates the service status cannot be determined
	ServiceStatusUnknown ServiceStatus = "UNKNOWN"
	// ServiceStatusNotFound indicates the requested service was not found
	ServiceStatusNotFound ServiceStatus = "NOT_FOUND"
)

// ServiceInfo contains metadata about a service
type ServiceInfo struct {
	Name          string        `json:"name"`
	Status        ServiceStatus `json:"status"`
	StartTime     time.Time     `json:"start_time,omitempty"`
	EventsHandled int64         `json:"events_handled,omitempty"`
	ActiveClients int           `json:"active_clients,omitempty"`
	ErrorCount    int           `json:"error_count,omitempty"`
	LastErrorTime time.Time     `json:"last_error_time,omitempty"`
}

// ActorService is the interface that all actor-based services implement
type ActorService interface {
	actor.Receiver
}

// ActorServiceManager manages multiple actor services
type ActorServiceManager struct {
	// Core components
	config            *internal.Config
	logger            *internal.Logger
	database          interfaces.DatabaseClient
	fireflyClient     interfaces.FireflyClient
	blockchainClients map[string]interfaces.BlockchainClient
	bankClients       []interfaces.BankAccountClient

	// Actor system components
	engine *actor.Engine
	ctx    context.Context
	cancel context.CancelFunc

	// Service registry
	services map[string]*actor.PID

	// Metadata about services
	serviceInfo map[string]*ServiceInfo
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

// StartMsg is the message sent to an actor to start it
type StartMsg struct{}

// StopMsg is the message sent to an actor to stop it
type StopMsg struct{}

// StatusRequestMsg is a message requesting the current status of an actor
type StatusRequestMsg struct{}

// StatusResponseMsg is the response to a status request
type StatusResponseMsg struct {
	Status      ServiceStatus
	LastActive  time.Time
	ErrorCount  int
	LastError   error
	CustomStats map[string]interface{}
}
