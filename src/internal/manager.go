package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/firefly"
	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/nats-io/nats.go"
)

// ManagedService defines a service that runs until its context is cancelled.
type ManagedService interface {
	Start(ctx context.Context) error
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

// ServiceManager manages multiple services and their lifecycles as background tasks.
type ServiceManager struct {
	// Service registry
	services    map[string]ManagedService
	cancels     map[string]context.CancelFunc
	serviceInfo map[string]*ServiceInfo
	startTimes  map[string]time.Time

	// Core components
	config          *Config
	logger          *Logger
	database        interfaces.DatabaseClient
	fireflyClient   interfaces.FireflyClient
	blockchainClients map[string]interfaces.BlockchainClient
	bankClients     []interfaces.BankAccountClient

	// Concurrency control
	mu     sync.Mutex
	wg     sync.WaitGroup
	rootCtx    context.Context
	rootCancel context.CancelFunc
}

// NewServiceManager creates a new service manager
func NewServiceManager(config *Config, logger *Logger) *ServiceManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ServiceManager{
		// Initialize service registry maps
		services:    make(map[string]ManagedService),
		cancels:     make(map[string]context.CancelFunc),
		serviceInfo: make(map[string]*ServiceInfo),
		startTimes:  make(map[string]time.Time),

		// Initialize core components
		config:          config,
		logger:          logger,
		blockchainClients: make(map[string]interfaces.BlockchainClient),
		bankClients:     []interfaces.BankAccountClient{},

		// Initialize concurrency control
		rootCtx:     ctx,
		rootCancel:  cancel,
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
	if (!exists) {
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
	if (!exists) {
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

// Start begins the service operations
func (m *ServiceManager) Start(runOnce bool) error {
	if runOnce {
		return m.runImportCycle()
	}
	
	// Parse interval duration
	interval, err := time.ParseDuration(m.config.Interval)
	if (err != nil) {
		return fmt.Errorf("invalid interval format: %w", err)
	}
	
	// Start background task
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		// Run immediately on startup
		if err := m.runImportCycle(); err != nil {
			m.logger.Error(ComponentGeneral, "Import cycle error: %v", err)
		}
		
		for {
			select {
			case <-ticker.C:
				if err := m.runImportCycle(); err != nil {
					m.logger.Error(ComponentGeneral, "Import cycle error: %v", err)
				}
			case <-m.rootCtx.Done():
				m.logger.Info(ComponentGeneral, "Stopping import cycle")
				return
			}
		}
	}()
	
	return nil
}

// Stop gracefully shuts down the service
func (m *ServiceManager) Stop() error {
	m.logger.Info(ComponentGeneral, "Stopping service manager")
	m.rootCancel()
	
	// Wait for all tasks to complete
	waitCh := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(waitCh)
	}()
	
	// Wait with timeout
	select {
	case <-waitCh:
		m.logger.Info(ComponentGeneral, "All tasks completed")
	case <-time.After(5 * time.Second):
		m.logger.Warn(ComponentGeneral, "Timed out waiting for tasks to complete")
	}
	
	// Close database connection
	if m.database != nil {
		if err := m.database.Close(); err != nil {
			m.logger.Error(ComponentGeneral, "Error closing database: %v", err)
		}
	}
	
	return nil
}

// runImportCycle performs a complete import cycle for all configured sources
func (m *ServiceManager) runImportCycle() error {
	m.logger.Info(ComponentGeneral, "Starting import cycle")
	
	// Import from blockchain wallets
	for chain, address := range m.config.Wallets {
		client, ok := m.blockchainClients[chain]
		if (!ok) {
			m.logger.Warn(ComponentGeneral, "No client found for %s", chain)
			continue
		}
		
		if err := m.importBlockchainTransactions(client, address); err != nil {
			m.logger.Error(ComponentGeneral, "Error importing %s transactions: %v", chain, err)
		}
	}
	
	// Import from bank accounts
	for _, client := range m.bankClients {
		if err := m.importBankTransactions(client); err != nil {
			m.logger.Error(ComponentGeneral, "Error importing bank transactions from %s: %v", 
				client.GetAccountName(), err)
		}
	}
	
	m.logger.Info(ComponentGeneral, "Import cycle completed")
	return nil
}

// importBlockchainTransactions imports transactions from a blockchain wallet
func (m *ServiceManager) importBlockchainTransactions(client interfaces.BlockchainClient, address string) error {
	m.logger.Info(ComponentGeneral, "Importing transactions from %s wallet: %s", 
		client.GetName(), address)
	
	// Get last import time
	lastImportTime, err := m.database.GetLastImportTime(client.GetName() + ":" + address)
	if err != nil {
		return fmt.Errorf("failed to get last import time: %w", err)
	}
	
	// Fetch transactions
	transactions, err := client.FetchTransactions(address)
	if err != nil {
		return fmt.Errorf("failed to fetch transactions: %w", err)
	}
	
	m.logger.Info(ComponentGeneral, "Fetched %d transactions from %s", 
		len(transactions), client.GetName())
	
	// Filter transactions by last import time
	var filtered []firefly.CustomTransaction
	for _, tx := range transactions {
		if lastImportTime.IsZero() || tx.Date.After(lastImportTime) {
			filtered = append(filtered, tx)
		}
	}
	
	if len(filtered) == 0 {
		m.logger.Info(ComponentGeneral, "No new transactions to import")
		return nil
	}
	
	m.logger.Info(ComponentGeneral, "Importing %d new transactions", len(filtered))
	
	// Import transactions
	importedCount := 0
	var lastTimestamp time.Time
	
	for _, tx := range filtered {
		// Check if already imported
		imported, err := m.database.IsTransactionImported(tx.ID)
		if err != nil {
			m.logger.Error(ComponentGeneral, "Error checking if transaction %s is imported: %v", 
				tx.ID, err)
			continue
		}
		
		if imported {
			m.logger.Debug(ComponentGeneral, "Transaction %s already imported, skipping", tx.ID)
			continue
		}
		
		 // Mark as imported
		metadata := map[string]string{
			"chain":       client.GetName(),
			"currency":    tx.Currency,
			"amount":      fmt.Sprintf("%f", tx.Amount),
			"type":        tx.TransType,
			"description": tx.Description,
			"timestamp":   tx.Date.Format(time.RFC3339),
		}
		
		if err := m.database.MarkTransactionAsImported(tx.ID, metadata); err != nil {
			m.logger.Error(ComponentGeneral, "Error marking transaction %s as imported: %v", 
				tx.ID, err)
			continue
		}
		
		if tx.Date.After(lastTimestamp) {
			lastTimestamp = tx.Date
		}
		
		importedCount++
	}
	
	m.logger.Info(ComponentGeneral, "Imported %d transactions from %s", 
		importedCount, client.GetName())
	
	// Update last import time
	if !lastTimestamp.IsZero() {
		if err := m.database.SetLastImportTime(client.GetName()+":"+address, lastTimestamp); err != nil {
			m.logger.Error(ComponentGeneral, "Error updating last import time: %v", err)
		}
	}
	
	return nil
}

// importBankTransactions imports transactions from a bank account
func (m *ServiceManager) importBankTransactions(client interfaces.BankAccountClient) error {
	m.logger.Info(ComponentGeneral, "Importing transactions from bank account: %s", 
		client.GetAccountName())
	
	// Get configuration for this bank
	var bankConfig *BankAccountConfig
	for i := range m.config.BankAccounts {
		if m.config.BankAccounts[i].Name == client.GetAccountName() {
			bankConfig = &m.config.BankAccounts[i]
			break
		}
	}
	
	if bankConfig == nil {
		return fmt.Errorf("bank account configuration not found for: %s", client.GetAccountName())
	}
	
	// Fetch balances first
	balances, err := client.FetchBalances()
	if err != nil {
		m.logger.Error(ComponentGeneral, "Error fetching balances: %v", err)
	} else {
		m.logger.Info(ComponentGeneral, "Fetched %d balances from %s", 
			len(balances), client.GetAccountName())
		
		// TODO: Update balances in Firefly III
	}
	
	// Get last import time
	lastImportTime, err := m.database.GetLastImportTime(client.GetName() + ":" + client.GetAccountName())
	if err != nil {
		return fmt.Errorf("failed to get last import time: %w", err)
	}
	
	// Set date range based on config and last import time
	fromDate := bankConfig.FromDate
	if fromDate == "" && !lastImportTime.IsZero() {
		fromDate = lastImportTime.Format("2006-01-02")
	}
	
	// Fetch transactions
	transactions, err := client.FetchTransactions(bankConfig.Limit, fromDate, bankConfig.ToDate)
	if err != nil {
		return fmt.Errorf("failed to fetch transactions: %w", err)
	}
	
	m.logger.Info(ComponentGeneral, "Fetched %d transactions from %s", 
		len(transactions), client.GetAccountName())
	
	if len(transactions) == 0 {
		m.logger.Info(ComponentGeneral, "No new transactions to import")
		return nil
	}
	
	// Import transactions
	importedCount := 0
	var lastTimestamp time.Time
	
	for _, tx := range transactions {
		// Check if already imported
		imported, err := m.database.IsTransactionImported(tx.ID)
		if err != nil {
			m.logger.Error(ComponentGeneral, "Error checking if transaction %s is imported: %v", 
				tx.ID, err)
			continue
		}
		
		if imported {
			m.logger.Debug(ComponentGeneral, "Transaction %s already imported, skipping", tx.ID)
			continue
		}
		
		 // Mark as imported
		metadata := map[string]string{
			"bank":        client.GetName(),
			"account":     client.GetAccountName(),
			"currency":    tx.Currency,
			"amount":      fmt.Sprintf("%f", tx.Amount),
			"type":        tx.TransType,
			"description": tx.Description,
			"timestamp":   tx.Date.Format(time.RFC3339),
		}
		
		if err := m.database.MarkTransactionAsImported(tx.ID, metadata); err != nil {
			m.logger.Error(ComponentGeneral, "Error marking transaction %s as imported: %v", 
				tx.ID, err)
			continue
		}
		
		if tx.Date.After(lastTimestamp) {
			lastTimestamp = tx.Date
		}
		
		importedCount++
	}
	
	m.logger.Info(ComponentGeneral, "Imported %d transactions from %s", 
		importedCount, client.GetAccountName())
	
	// Update last import time
	if !lastTimestamp.IsZero() {
		if err := m.database.SetLastImportTime(
			client.GetName()+":"+client.GetAccountName(), lastTimestamp); err != nil {
			m.logger.Error(ComponentGeneral, "Error updating last import time: %v", err)
		}
	}
	
	return nil
}
