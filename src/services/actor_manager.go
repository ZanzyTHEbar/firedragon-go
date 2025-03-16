package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/adapters/banking"
	"github.com/ZanzyTHEbar/firedragon-go/adapters/blockchain"
	"github.com/ZanzyTHEbar/firedragon-go/firefly"
	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/anthdm/hollywood/actor"
	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
)

// NewActorServiceManager creates a new actor-based service manager
func NewActorServiceManager(config *internal.Config, logger *internal.Logger) *ActorServiceManager {
	ctx, cancel := context.WithCancel(context.Background())

	// Create the actor engine with default config
	engine := actor.NewEngine(actor.NewEngineConfig())

	return &ActorServiceManager{
		config:            config,
		logger:            logger,
		blockchainClients: make(map[string]interfaces.BlockchainClient),
		bankClients:       []interfaces.BankAccountClient{},
		ctx:               ctx,
		cancel:            cancel,
		services:          make(map[string]*actor.PID),
		serviceInfo:       make(map[string]*ServiceInfo),
		engine:            engine,
	}
}

// Initialize sets up all components needed for the service manager
func (m *ActorServiceManager) Initialize() error {
	// Initialize database
	db, err := NewSQLiteDatabase(m.config.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	m.database = db

	// Initialize Firefly III client
	fireflyClient, err := firefly.New(m.config.Firefly.URL, m.config.Firefly.Token)
	if err != nil {
		return fmt.Errorf("failed to initialize Firefly client: %w", err)
	}
	m.fireflyClient = fireflyClient

	// Initialize blockchain clients
	for chain := range m.config.Wallets {
		if client := blockchain.NewClient(chain); client != nil {
			m.blockchainClients[chain] = client
			m.logger.Info(internal.ComponentGeneral, "Initialized %s client", chain)
		} else {
			m.logger.Warn(internal.ComponentGeneral, "Unsupported blockchain: %s", chain)
		}
	}

	// Initialize bank account clients
	for _, bankConfig := range m.config.BankAccounts {
		if client := banking.NewClient(
			bankConfig.Provider,
			bankConfig.Name,
			bankConfig.Credentials["client_id"],
			bankConfig.Credentials["client_secret"],
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

	// Register the actor with the engine and get its PID
	props := actor.PropsFromProducer(func() actor.Receiver {
		return transactionActor
	})
	transactionPID := m.engine.Spawn(props, "transaction_service")
	m.services["transaction_service"] = transactionPID

	// Initialize service info
	m.serviceInfo["transaction_service"] = &ServiceInfo{
		Name:   "transaction_service",
		Status: ServiceStatusStopped,
	}

	return nil
}

// Register adds a new actor service with a unique name to the manager
func (m *ActorServiceManager) Register(name string, service ActorService) {
	m.logger.Debug(internal.ComponentService, "Registering actor service: %s", name)

	// Register the actor with the engine and get its PID
	props := actor.PropsFromProducer(func() actor.Receiver {
		return service
	})
	pid := m.engine.Spawn(props, name)
	m.services[name] = pid

	// Initialize service info
	m.serviceInfo[name] = &ServiceInfo{
		Name:   name,
		Status: ServiceStatusStopped,
	}

	m.logger.Debug(internal.ComponentService, "Actor service %s registered successfully", name)
}

// StartService starts a specific actor service by name
func (m *ActorServiceManager) StartService(name string) error {
	m.logger.Debug(internal.ComponentService, "StartService called for actor service: %s", name)

	pid, exists := m.services[name]
	if !exists {
		return fmt.Errorf("actor service %s not found", name)
	}

	// Update service info
	if info, ok := m.serviceInfo[name]; ok {
		info.Status = ServiceStatusRunning
		info.StartTime = time.Now()
	}

	// Send start message to actor
	m.engine.Send(pid, StartMsg{})

	m.logger.Debug(internal.ComponentService, "Actor service %s started", name)
	return nil
}

// StopService stops a specific actor service by name
func (m *ActorServiceManager) StopService(name string) error {
	m.logger.Debug(internal.ComponentService, "StopService called for actor service: %s", name)

	pid, exists := m.services[name]
	if !exists {
		return fmt.Errorf("actor service %s not found", name)
	}

	// Update service info
	if info, ok := m.serviceInfo[name]; ok {
		info.Status = ServiceStatusStopped
	}

	// Send stop message to actor
	m.engine.Send(pid, StopMsg{})

	m.logger.Debug(internal.ComponentService, "Actor service %s stopped", name)
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
				info.Status = ServiceStatusRunning
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
				info.Status = ServiceStatusStopped
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
func (m *ActorServiceManager) GetServiceInfo(name string) (*ServiceInfo, error) {
	pid, exists := m.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	// TODO: we want to ask the actor for its current status
	info, exists := m.serviceInfo[name]
	if !exists {
		return nil, fmt.Errorf("service info for %s not found", name)
	}

	// Create a copy to avoid concurrent modifications
	infoCopy := *info
	return &infoCopy, nil
}

// GetAllServicesInfo returns information about all registered services
func (m *ActorServiceManager) GetAllServicesInfo() []*ServiceInfo {
	result := make([]*ServiceInfo, 0, len(m.serviceInfo))

	for name := range m.services {
		info, err := m.GetServiceInfo(name)
		if err != nil {
			m.logger.Warn(internal.ComponentService, "Failed to get info for service %s: %v", name, err)
			continue
		}
		result = append(result, info)
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
func publishActorServiceStatus(conn *nats.Conn, services []*ServiceInfo, log *internal.Logger) {
	data, err := json.Marshal(services)
	if err != nil {
		log.Error(internal.ComponentService, "Error marshaling service status: %v", err)
		return
	}

	if err := conn.Publish("service.status", data); err != nil {
		log.Error(internal.ComponentService, "Error publishing service status: %v", err)
	}
}
