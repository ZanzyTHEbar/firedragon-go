package services

import (
	"fmt"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/adapters"
	"github.com/ZanzyTHEbar/firedragon-go/firefly"
	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
)

// Initialize sets up all components needed for the service
func (m *ServiceManager) Initialize() error {
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
		if client := adapters.NewBlockchainClient(chain); client != nil {
			m.blockchainClients[chain] = client
			m.logger.Info(ComponentGeneral, "Initialized %s client", chain)
		} else {
			m.logger.Warn(ComponentGeneral, "Unsupported blockchain: %s", chain)
		}
	}

	// Initialize bank account clients
	for _, bankConfig := range m.config.BankAccounts {
		if client := adapters.NewBankingClient(
			bankConfig.Provider,
			bankConfig.Name,
			bankConfig.Credentials["client_id"],
			bankConfig.Credentials["client_secret"],
		); client != nil {
			m.bankClients = append(m.bankClients, client)
			m.logger.Info(ComponentGeneral, "Initialized bank client: %s", bankConfig.Name)
		} else {
			m.logger.Warn(ComponentGeneral, "Unsupported bank provider: %s", bankConfig.Provider)
		}
	}

	// Initialize and register transaction service
	transactionService := services.NewTransactionService(
		m.blockchainClients["ethereum"], // Use first blockchain client
		m.bankClients[0],                // Use first bank client
		m.fireflyClient,
		m.database,
		m.logger,
		m.config,
	)
	m.Register("transaction_service", transactionService)

	return nil
}

// runImportCycle performs a complete import cycle for all configured sources
func (m *ServiceManager) runImportCycle() error {
	m.logger.Info(internal.ComponentService, "Starting import cycle")

	// Import from blockchain wallets
	for chain, address := range m.config.Wallets {
		client, ok := m.blockchainClients[chain]
		if !ok {
			m.logger.Warn(internal.ComponentService, "No client found for %s", chain)
			continue
		}

		if err := m.importBlockchainTransactions(client, address); err != nil {
			m.logger.Error(internal.ComponentService, "Error importing %s transactions: %v", chain, err)
		}
	}

	// Import from bank accounts
	for _, client := range m.bankClients {
		if err := m.importBankTransactions(client); err != nil {
			m.logger.Error(internal.ComponentService, "Error importing bank transactions from %s: %v",
				client.GetAccountName(), err)
		}
	}

	m.logger.Info(internal.ComponentService, "Import cycle completed")
	return nil
}

// importBlockchainTransactions imports transactions from a blockchain wallet
func (m *ServiceManager) importBlockchainTransactions(client interfaces.BlockchainClient, address string) error {
	m.logger.Info(internal.ComponentService, "Importing transactions from %s wallet: %s",
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

	m.logger.Info(internal.ComponentService, "Fetched %d transactions from %s",
		len(transactions), client.GetName())

	// Filter transactions by last import time
	var filtered []firefly.CustomTransaction
	for _, tx := range transactions {
		if lastImportTime.IsZero() || tx.Date.After(lastImportTime) {
			filtered = append(filtered, tx)
		}
	}

	if len(filtered) == 0 {
		m.logger.Info(internal.ComponentService, "No new transactions to import")
		return nil
	}

	m.logger.Info(internal.ComponentService, "Importing %d new transactions", len(filtered))

	// Import transactions
	importedCount := 0
	var lastTimestamp time.Time

	for _, tx := range filtered {
		// Check if already imported
		imported, err := m.database.IsTransactionImported(tx.ID)
		if err != nil {
			m.logger.Error(internal.ComponentService, "Error checking if transaction %s is imported: %v",
				tx.ID, err)
			continue
		}

		if imported {
			m.logger.Debug(internal.ComponentService, "Transaction %s already imported, skipping", tx.ID)
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
			m.logger.Error(internal.ComponentService, "Error marking transaction %s as imported: %v",
				tx.ID, err)
			continue
		}

		if tx.Date.After(lastTimestamp) {
			lastTimestamp = tx.Date
		}

		importedCount++
	}

	m.logger.Info(internal.ComponentService, "Imported %d transactions from %s",
		importedCount, client.GetName())

	// Update last import time
	if !lastTimestamp.IsZero() {
		if err := m.database.SetLastImportTime(client.GetName()+":"+address, lastTimestamp); err != nil {
			m.logger.Error(internal.ComponentService, "Error updating last import time: %v", err)
		}
	}

	return nil
}

// importBankTransactions imports transactions from a bank account
func (m *ServiceManager) importBankTransactions(client interfaces.BankAccountClient) error {
	m.logger.Info(internal.ComponentService, "Importing transactions from bank account: %s",
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
		m.logger.Error(internal.ComponentService, "Error fetching balances: %v", err)
	} else {
		m.logger.Info(internal.ComponentService, "Fetched %d balances from %s",
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

	m.logger.Info(internal.ComponentService, "Fetched %d transactions from %s",
		len(transactions), client.GetAccountName())

	if len(transactions) == 0 {
		m.logger.Info(internal.ComponentService, "No new transactions to import")
		return nil
	}

	// Import transactions
	importedCount := 0
	var lastTimestamp time.Time

	for _, tx := range transactions {
		// Check if already imported
		imported, err := m.database.IsTransactionImported(tx.ID)
		if err != nil {
			m.logger.Error(internal.ComponentService, "Error checking if transaction %s is imported: %v",
				tx.ID, err)
			continue
		}

		if imported {
			m.logger.Debug(internal.ComponentService, "Transaction %s already imported, skipping", tx.ID)
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
			m.logger.Error(internal.ComponentService, "Error marking transaction %s as imported: %v",
				tx.ID, err)
			continue
		}

		if tx.Date.After(lastTimestamp) {
			lastTimestamp = tx.Date
		}

		importedCount++
	}

	m.logger.Info(internal.ComponentService, "Imported %d transactions from %s",
		importedCount, client.GetAccountName())

	// Update last import time
	if !lastTimestamp.IsZero() {
		if err := m.database.SetLastImportTime(
			client.GetName()+":"+client.GetAccountName(), lastTimestamp); err != nil {
			m.logger.Error(internal.ComponentService, "Error updating last import time: %v", err)
		}
	}

	return nil
}
