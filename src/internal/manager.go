package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// ManagedService defines a service that runs until its context is cancelled.
type ManagedService interface {
	Start(ctx context.Context) error
}

// ServiceManager manages multiple services and their lifecycles as background tasks.
type ServiceManager struct {
	services    map[string]ManagedService
	cancels     map[string]context.CancelFunc
	mu          sync.Mutex
	rootCtx     context.Context
	rootCancel  context.CancelFunc
	serviceInfo map[string]*ServiceInfo // Track info for each service
	startTimes  map[string]time.Time    // Track when each service was started
}

type ServiceStatus string

const (
	ServiceStatusRunning  ServiceStatus = "RUNNING"
	ServiceStatusStopped  ServiceStatus = "STOPPED"
	ServiceStatusError    ServiceStatus = "ERROR"
	ServiceStatusUnknown  ServiceStatus = "UNKNOWN"
	ServiceStatusNotFound ServiceStatus = "NOT_FOUND"
)

type ServiceInfo struct {
	Name          string        `json:"name"`
	Status        ServiceStatus `json:"status"`
	StartTime     time.Time     `json:"start_time,omitempty"`
	EventsHandled int64         `json:"events_handled,omitempty"`
	ActiveClients int           `json:"active_clients,omitempty"`
	ErrorCount    int           `json:"error_count,omitempty"`
	LastErrorTime time.Time     `json:"last_error_time,omitempty"`
}

// NewServiceManagerWithContext creates a new ServiceManager with a provided parent context.
func NewServiceManager(parentCtx context.Context) *ServiceManager {
	rootCtx, rootCancel := context.WithCancel(parentCtx)
	return &ServiceManager{
		services:    make(map[string]ManagedService),
		cancels:     make(map[string]context.CancelFunc),
		rootCtx:     rootCtx,
		rootCancel:  rootCancel,
		serviceInfo: make(map[string]*ServiceInfo),
		startTimes:  make(map[string]time.Time),
	}
}

// Register adds a new service with a unique name to the manager.
func (m *ServiceManager) Register(name string, service ManagedService) {
	logger := GetLogger()
	logger.Debug(ComponentGeneral, "Registering service: %s", name)

	if service == nil {
		logger.Error(ComponentGeneral, "Attempted to register nil service: %s", name)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.services[name] = service

	// Initialize service info
	m.serviceInfo[name] = &ServiceInfo{
		Name:   name,
		Status: ServiceStatusStopped,
	}

	logger.Debug(ComponentGeneral, "Service %s registered successfully", name)
}

// StartService starts a specific service by name in the background.
func (m *ServiceManager) StartService(name string) error {
	logger := GetLogger()
	logger.Debug(ComponentGeneral, "StartService called for service: %s", name)

	m.mu.Lock()
	defer m.mu.Unlock()

	svc, exists := m.services[name]
	if !exists {
		logger.Error(ComponentGeneral, "Service %s not found", name)
		return fmt.Errorf("service %s not found", name)
	}

	if _, running := m.cancels[name]; running {
		logger.Error(ComponentGeneral, "Service %s is already running", name)
		return fmt.Errorf("service %s is already running", name)
	}

	// Create a child context of the root context
	ctx, cancel := context.WithCancel(m.rootCtx)
	m.cancels[name] = cancel

	// Update service info
	m.serviceInfo[name].Status = ServiceStatusRunning
	m.serviceInfo[name].StartTime = time.Now()
	m.startTimes[name] = time.Now()

	// Start service in goroutine but wait for initialization
	errChan := make(chan error, 1)

	logger.Debug(ComponentGeneral, "Launching service goroutine for %s", name)

	go func() {
		logger.Debug(ComponentGeneral, "Service goroutine started for %s", name)

		// log service start
		logger.Info(ComponentGeneral, "Starting service %s", name)

		if err := svc.Start(ctx); err != nil {
			logger.Error(ComponentGeneral, "Service %s error: %v", name, err)
			errChan <- err

			m.mu.Lock()
			if info, ok := m.serviceInfo[name]; ok {
				info.Status = ServiceStatusError
				info.ErrorCount++
				info.LastErrorTime = time.Now()
			}
			m.mu.Unlock()
		}

		// log service stop
		logger.Info(ComponentGeneral, "Service %s stopped", name)

		m.mu.Lock()
		delete(m.cancels, name)
		if info, ok := m.serviceInfo[name]; ok {
			info.Status = ServiceStatusStopped
		}
		m.mu.Unlock()

		close(errChan)
	}()

	// Wait briefly for any immediate startup errors
	select {
	case err := <-errChan:
		if err != nil {
			logger.Error(ComponentGeneral, "Service %s failed to start: %v", name, err)
			return fmt.Errorf("service %s failed to start: %w", name, err)
		}
	case <-time.After(100 * time.Millisecond):
		logger.Debug(ComponentGeneral, "Service %s started successfully", name)
	}

	return nil
}

// StopService stops a specific service by name.
func (m *ServiceManager) StopService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cancel, exists := m.cancels[name]
	if !exists {
		return fmt.Errorf("service %s is not running", name)
	}
	cancel()
	delete(m.cancels, name)

	// Update service status
	if info, ok := m.serviceInfo[name]; ok {
		info.Status = ServiceStatusStopped
	}

	return nil
}

// StartAll starts all registered services in the background.
func (m *ServiceManager) StartAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name := range m.services {
		if _, running := m.cancels[name]; running {
			continue
		}

		// Create a child context of the root context
		ctx, cancel := context.WithCancel(m.rootCtx)
		m.cancels[name] = cancel

		// Update service info
		m.serviceInfo[name].Status = ServiceStatusRunning
		m.serviceInfo[name].StartTime = time.Now()
		m.startTimes[name] = time.Now()

		go func(n string) {
			svc := m.services[n]
			if err := svc.Start(ctx); err != nil {
				GetLogger().Error(ComponentGeneral, "Service %s error: %v", n, err)

				m.mu.Lock()
				if info, ok := m.serviceInfo[n]; ok {
					info.Status = ServiceStatusError
					info.ErrorCount++
					info.LastErrorTime = time.Now()
				}
				m.mu.Unlock()
			}

			m.mu.Lock()
			delete(m.cancels, n)
			if info, ok := m.serviceInfo[n]; ok {
				info.Status = ServiceStatusStopped
			}
			m.mu.Unlock()
		}(name)
	}
	return nil
}

// StopAll stops all running services.
func (m *ServiceManager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, cancel := range m.cancels {
		cancel()
		delete(m.cancels, name)

		// Update service status
		if info, ok := m.serviceInfo[name]; ok {
			info.Status = ServiceStatusStopped
		}
	}
	return nil
}

// Shutdown stops all services and cancels the root context
func (m *ServiceManager) Shutdown() error {
	m.StopAll()
	m.rootCancel()
	return nil
}

// GetRootContext returns the root context of the service manager
func (m *ServiceManager) GetRootContext() context.Context {
	return m.rootCtx
}

func (m *ServiceManager) GetService(name string) (ManagedService, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	svc, exists := m.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}
	return svc, nil
}

func (m *ServiceManager) GetServices() map[string]ManagedService {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.services
}

// GetServiceInfo returns information about a specific service
func (m *ServiceManager) GetServiceInfo(name string) (*ServiceInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.serviceInfo[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	// Create a copy to avoid concurrent modifications
	infoCopy := *info
	return &infoCopy, nil
}

// GetAllServicesInfo returns information about all registered services
func (m *ServiceManager) GetAllServicesInfo() []*ServiceInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*ServiceInfo, 0, len(m.serviceInfo))
	for _, info := range m.serviceInfo {
		// Create copies to avoid concurrent modifications
		infoCopy := *info
		result = append(result, &infoCopy)
	}

	return result
}

// UpdateServiceStats updates the statistics for a service
func (m *ServiceManager) UpdateServiceStats(name string, eventsHandled int64, activeClients int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.serviceInfo[name]
	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	info.EventsHandled = eventsHandled
	info.ActiveClients = activeClients

	return nil
}

// RecordServiceError records an error for the specified service
func (m *ServiceManager) RecordServiceError(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, exists := m.serviceInfo[name]; exists {
		info.ErrorCount++
		info.LastErrorTime = time.Now()
	}
}

// StartServices sets up NATS subscriptions to control services
func StartServices(natsChannel string, conn *nats.Conn, mgr *ServiceManager, log *Logger) {
	// Subscribe to service control commands
	_, err := conn.Subscribe(natsChannel, func(m *nats.Msg) {
		event := string(m.Data)
		log.Info(ComponentGeneral, "Received event: %s", event)
		if strings.HasPrefix(event, "start:") {
			serviceName := strings.TrimPrefix(event, "start:")
			if err := mgr.StartService(serviceName); err != nil {
				log.Error(ComponentGeneral, "Error starting service %s: %v", serviceName, err)
			} else {
				log.Info(ComponentGeneral, "Service %s started successfully", serviceName)
			}
		} else if strings.HasPrefix(event, "stop:") {
			serviceName := strings.TrimPrefix(event, "stop:")
			if err := mgr.StopService(serviceName); err != nil {
				log.Error(ComponentGeneral, "Error stopping service %s: %v", serviceName, err)
			} else {
				log.Info(ComponentGeneral, "Service %s stopped successfully", serviceName)
			}
		} else if event == "status" {
			// Get status of all services and publish it
			services := mgr.GetAllServicesInfo()
			publishServiceStatus(conn, services, log)
		} else {
			log.Info(ComponentGeneral, "Unknown event: %s", event)
		}
	})
	if err != nil {
		log.Fatal(ComponentGeneral, "Error subscribing to events: %v", err)
	}

	// Set up periodic status updates (every 30 seconds)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				services := mgr.GetAllServicesInfo()
				publishServiceStatus(conn, services, log)
			case <-mgr.GetRootContext().Done():
				return
			}
		}
	}()
}

// publishServiceStatus publishes service status information via NATS
func publishServiceStatus(conn *nats.Conn, services []*ServiceInfo, log *Logger) {
	data, err := json.Marshal(services)
	if err != nil {
		log.Error(ComponentGeneral, "Error marshaling service status: %v", err)
		return
	}

	if err := conn.Publish("service.status", data); err != nil {
		log.Error(ComponentGeneral, "Error publishing service status: %v", err)
	}
}
