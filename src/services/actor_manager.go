package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/factory"
	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/anthdm/hollywood/actor"
	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
)

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
	services    map[string]*actor.PID
	serviceInfo map[string]interfaces.ServiceInfo
	mu          sync.RWMutex
	natsClient  *nats.Conn
}

// NewActorServiceManager creates a new actor-based service manager
func NewActorServiceManager(config *internal.Config, logger *internal.Logger) ActorServiceManager {
	ctx, cancel := context.WithCancel(context.Background())

	// Create the actor engine with default config
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		logger.Error(internal.ComponentGeneral, "Failed to create actor engine: %v", err)
		cancel()
		return ActorServiceManager{}
	}

	return ActorServiceManager{
		config:            config,
		logger:            logger,
		blockchainClients: make(map[string]interfaces.BlockchainClient),
		bankClients:       []interfaces.BankAccountClient{},
		ctx:               ctx,
		cancel:            cancel,
		services:          make(map[string]*actor.PID),
		serviceInfo:       make(map[string]interfaces.ServiceInfo),
		engine:            engine,
	}
}

// Initialize sets up all components needed for the service manager
func (m *ActorServiceManager) Initialize() error {
	// Initialize database
	db, err := factory.NewDatabaseClient(m.config.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	m.database = db

	// Initialize Firefly III client
	fireflyClient, err := factory.NewFireflyClient(m.config.Firefly.URL, m.config.Firefly.Token)
	if err != nil {
		return fmt.Errorf("failed to initialize Firefly client: %w", err)
	}
	m.fireflyClient = fireflyClient

	// Initialize blockchain clients
	for chain := range m.config.Wallets {
		if client := factory.NewBlockchainClient(chain); client != nil {
			m.blockchainClients[chain] = client
			m.logger.Info(internal.ComponentGeneral, "Initialized %s client", chain)
		} else {
			m.logger.Warn(internal.ComponentGeneral, "Unsupported blockchain: %s", chain)
		}
	}

	// Initialize bank account clients
	for _, bankConfig := range m.config.BankAccounts {
		if client := factory.NewBankingClient(
			bankConfig.Provider,
			bankConfig.Name,
			bankConfig.Credentials,
		); client != nil {
			m.bankClients = append(m.bankClients, client)
			m.logger.Info(internal.ComponentGeneral, "Initialized bank client: %s", bankConfig.Name)
		} else {
			m.logger.Warn(internal.ComponentGeneral, "Unsupported bank provider: %s", bankConfig.Provider)
		}
	}

	// Initialize and register transaction actor
	transactionActor := NewTransactionActor(
		m.blockchainClients,
		m.bankClients,
		m.fireflyClient,
		m.database,
		m.logger,
		m.config,
	)

	// Register the actor with the engine
	transactionPID := m.engine.SpawnFunc(func(ctx *actor.Context) {
		transactionActor.Receive(ctx)
	}, "transaction_service")
	m.services["transaction_service"] = transactionPID

	// Initialize service info
	m.serviceInfo["transaction_service"] = interfaces.ServiceInfo{
		Name:   "transaction_service",
		Status: interfaces.ServiceStatusStopped,
	}

	return nil
}

// Register adds a new actor service with a unique name to the manager
func (m *ActorServiceManager) Register(name string, service ActorService) {
	m.logger.Debug(internal.ComponentService, "Registering actor service: %s", name)

	// Register the actor with the engine
	pid := m.engine.SpawnFunc(func(ctx *actor.Context) {
		service.Receive(ctx)
	}, name)
	m.services[name] = pid

	// Initialize service info
	m.serviceInfo[name] = interfaces.ServiceInfo{
		Name:   name,
		Status: interfaces.ServiceStatusStopped,
	}

	m.logger.Debug(internal.ComponentService, "Actor service %s registered successfully", name)
}

// StartService starts a specific actor service by name
func (m *ActorServiceManager) StartService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pid, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	m.engine.Send(pid, &StartMsg{})
	return nil
}

// StopService stops a specific actor service by name
func (m *ActorServiceManager) StopService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pid, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	m.engine.Send(pid, &StopMsg{})
	return nil
}

// StartAll starts all registered actor services
func (m *ActorServiceManager) StartAll() error {
	m.logger.Debug(internal.ComponentService, "Starting all actor services")

	var g errgroup.Group
	var mu sync.Mutex

	for name, pid := range m.services {
		// Capture loop variables
		currentName := name
		currentPID := pid

		g.Go(func() error {
			// Update service info
			mu.Lock()
			if info, ok := m.serviceInfo[currentName]; ok {
				info.Status = interfaces.ServiceStatusRunning
				info.StartTime = time.Now()
			}
			mu.Unlock()

			// Send start message to actor
			m.engine.Send(currentPID, StartMsg{})

			m.logger.Debug(internal.ComponentService, "Actor service %s started", currentName)
			return nil
		})
	}

	return g.Wait()
}

// StopAll stops all running actor services
func (m *ActorServiceManager) StopAll() error {
	m.logger.Debug(internal.ComponentService, "Stopping all actor services")

	var g errgroup.Group
	var mu sync.Mutex

	for name, pid := range m.services {
		// Capture loop variables
		currentName := name
		currentPID := pid

		g.Go(func() error {
			// Update service info
			mu.Lock()
			if info, ok := m.serviceInfo[currentName]; ok {
				info.Status = interfaces.ServiceStatusStopped
			}
			mu.Unlock()

			// Send stop message to actor
			m.engine.Send(currentPID, StopMsg{})

			m.logger.Debug(internal.ComponentService, "Actor service %s stopped", currentName)
			return nil
		})
	}

	return g.Wait()
}

// Shutdown stops all services and shuts down the actor engine
func (m *ActorServiceManager) Shutdown() error {
	m.logger.Info(internal.ComponentService, "Shutting down actor service manager")

	// Stop all services
	if err := m.StopAll(); err != nil {
		m.logger.Error(internal.ComponentService, "Error stopping services: %v", err)
	}

	// Cancel root context
	m.cancel()

	// Close database connection
	if m.database != nil {
		if err := m.database.Close(); err != nil {
			m.logger.Error(internal.ComponentService, "Error closing database: %v", err)
		}
	}

	return nil
}

// Start begins the service operations
func (m *ActorServiceManager) Start(runOnce bool) error {
	if runOnce {
		return m.runImportCycle()
	}

	// Parse interval duration - we don't use this directly but it's good to validate
	_, err := time.ParseDuration(m.config.Interval)
	if err != nil {
		return fmt.Errorf("invalid interval format: %w", err)
	}

	// Start the transaction service
	if err := m.StartService("transaction_service"); err != nil {
		return fmt.Errorf("error starting transaction service: %w", err)
	}

	// Since we're using actors, we don't need to manually run the import cycle
	// as the transaction actor will handle this with its own timer
	// Just log that we've started
	m.logger.Info(internal.ComponentService, "Actor service manager started")

	return nil
}

// runImportCycle performs a one-time import cycle for all configured sources
func (m *ActorServiceManager) runImportCycle() error {
	m.logger.Info(internal.ComponentService, "Starting one-time import cycle")

	// Get the transaction service PID
	pid, exists := m.services["transaction_service"]
	if !exists {
		return fmt.Errorf("transaction service not found")
	}

	// Send a process request to the transaction actor
	m.engine.Send(pid, ProcessRequest{})

	// Since we're doing a one-time run, we'll just consider it successful
	// The actual results will be logged by the actor
	m.logger.Info(internal.ComponentService, "Import cycle initiated")

	return nil
}

// GetServiceInfo returns information about a specific service
func (m *ActorServiceManager) GetServiceInfo(name string) (*interfaces.ServiceInfo, error) {
	if _, exists := m.services[name]; !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	info, exists := m.serviceInfo[name]
	if !exists {
		return nil, fmt.Errorf("service info for %s not found", name)
	}

	pid, exists := m.services[name]
	if exists {
		// Send status request to actor
		future := m.engine.Request(pid, &StatusRequestMsg{}, 5*time.Second)
		if result, err := future.Result(); err == nil {
			if response, ok := result.(StatusResponseMsg); ok {
				info.Status = response.Status
				info.StartTime = response.LastActive
				info.ErrorCount = response.ErrorCount
				info.LastError = response.LastError
				info.CustomStats = response.CustomStats
			}
		}
	}

	// Create a copy to avoid concurrent modifications
	infoCopy := info
	return &infoCopy, nil
}

// GetAllServicesInfo returns information about all registered services
func (m *ActorServiceManager) GetAllServicesInfo() []*interfaces.ServiceInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*interfaces.ServiceInfo, 0, len(m.serviceInfo))

	for name, info := range m.serviceInfo {
		// Create a copy of the info to avoid concurrent modifications
		infoCopy := &interfaces.ServiceInfo{
			Name:          info.Name,
			Status:        info.Status,
			LastError:     info.LastError,
			StartTime:     info.StartTime,
			EventsHandled: info.EventsHandled,
			ActiveClients: info.ActiveClients,
			ErrorCount:    info.ErrorCount,
			LastErrorTime: info.LastErrorTime,
			CustomStats:   info.CustomStats,
		}

		// Get latest status from actor if available
		if pid, exists := m.services[name]; exists {
			m.engine.Send(pid, &StatusRequestMsg{})
			// The actor will update the service info asynchronously
		}

		result = append(result, infoCopy)
	}

	return result
}

// StartActorServices sets up NATS subscriptions to control actor services
func StartActorServices(natsChannel string, conn *nats.Conn, mgr *ActorServiceManager, log *internal.Logger) {
	// Subscribe to service control commands
	_, err := conn.Subscribe(natsChannel, func(m *nats.Msg) {
		event := string(m.Data)
		log.Info(internal.ComponentService, "Received event: %s", event)

		if strings.HasPrefix(event, "start:") {
			serviceName := strings.TrimPrefix(event, "start:")
			if err := mgr.StartService(serviceName); err != nil {
				log.Error(internal.ComponentService, "Error starting service %s: %v", serviceName, err)
			} else {
				log.Info(internal.ComponentService, "Service %s started successfully", serviceName)
			}
		} else if strings.HasPrefix(event, "stop:") {
			serviceName := strings.TrimPrefix(event, "stop:")
			if err := mgr.StopService(serviceName); err != nil {
				log.Error(internal.ComponentService, "Error stopping service %s: %v", serviceName, err)
			} else {
				log.Info(internal.ComponentService, "Service %s stopped successfully", serviceName)
			}
		} else if event == "status" {
			// Get status of all services and publish it
			services := mgr.GetAllServicesInfo()
			publishActorServiceStatus(conn, services, log)
		} else {
			log.Info(internal.ComponentService, "Unknown event: %s", event)
		}
	})

	if err != nil {
		log.Fatal(internal.ComponentService, "Error subscribing to events: %v", err)
	}

	// Set up periodic status updates (every 30 seconds)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				services := mgr.GetAllServicesInfo()
				publishActorServiceStatus(conn, services, log)
			case <-mgr.ctx.Done():
				return
			}
		}
	}()
}

// publishActorServiceStatus publishes service status information via NATS
func publishActorServiceStatus(conn *nats.Conn, services []*interfaces.ServiceInfo, log *internal.Logger) {
	data, err := json.Marshal(services)
	if err != nil {
		log.Error(internal.ComponentService, "Error marshaling service status: %v", err)
		return
	}

	if err := conn.Publish("service.status", data); err != nil {
		log.Error(internal.ComponentService, "Error publishing service status: %v", err)
	}
}

// Status field is used by publishActorServiceStatus
func (m *ActorServiceManager) publishActorServiceStatus() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, info := range m.serviceInfo {
		// Use Status and StartTime fields for NATS status updates
		status := &interfaces.ServiceInfo{
			Name:      name,
			Status:    info.Status,
			StartTime: info.StartTime,
			LastError: info.LastError,
		}

		// Marshal the status to JSON
		data, err := json.Marshal(status)
		if err != nil {
			m.logger.Error(internal.ComponentGeneral, "Failed to marshal service status: %v", err)
			continue
		}
		m.logger.Debug(internal.ComponentGeneral, "Publishing service status: %s", string(data))

		// Publish status via NATS
		if err := m.natsClient.Publish("services.status", data); err != nil {
			m.logger.Error(internal.ComponentGeneral, "Failed to publish service status: %v", err)
		}
	}
}

// UpdateServiceStatus updates a service's status and triggers status publishing
func (m *ActorServiceManager) UpdateServiceStatus(name string, status string) {
	m.mu.Lock()
	if info, exists := m.serviceInfo[name]; exists {
		info.Status = status
		if status == "started" {
			info.StartTime = time.Now()
		}
	}
	m.mu.Unlock()

	// Trigger status update
	m.publishActorServiceStatus()
}
