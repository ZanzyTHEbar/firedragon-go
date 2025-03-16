package interfaces

import (
	"time"
)

// ServiceStatus represents the current status of a service
type ServiceStatus = string

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
	LastError     error         `json:"last_error,omitempty"`
	StartTime     time.Time     `json:"start_time,omitempty"`
	EventsHandled int64         `json:"events_handled,omitempty"`
	ActiveClients int           `json:"active_clients,omitempty"`
	ErrorCount    int           `json:"error_count,omitempty"`
	LastErrorTime time.Time     `json:"last_error_time,omitempty"`
	CustomStats   interface{}   `json:"custom_stats,omitempty"`
}

// ServiceManager defines the interface for managing services
type ServiceManager interface {
	Initialize() error
	StartService(name string) error
	StopService(name string) error
	StartAll() error
	StopAll() error
	Shutdown() error
	Start(runOnce bool) error
	GetServiceInfo(name string) (*ServiceInfo, error)
	GetAllServicesInfo() []*ServiceInfo
}
