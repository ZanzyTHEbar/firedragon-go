package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
)

// TransactionService coordinates transaction processing between different components
type TransactionService struct {
	blockchainClient interfaces.BlockchainClient
	bankClient       interfaces.BankAccountClient
	fireflyClient    interfaces.FireflyClient
	db               interfaces.DatabaseClient
	logger           *internal.Logger
	config           *internal.Config
}

// NewTransactionService creates a new transaction service
func NewTransactionService(
	blockchainClient interfaces.BlockchainClient,
	bankClient interfaces.BankAccountClient,
	fireflyClient interfaces.FireflyClient,
	db interfaces.DatabaseClient,
	logger *internal.Logger,
	config *internal.Config,
) *TransactionService {
	return &TransactionService{
		blockchainClient: blockchainClient,
		bankClient:       bankClient,
		fireflyClient:    fireflyClient,
		db:               db,
		logger:           logger,
		config:           config,
	}
}

// Start implements the ManagedService interface
func (s *TransactionService) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.parseInterval(s.config.Interval))
	defer ticker.Stop()

	// Do initial import
	if err := s.ProcessTransactions(); err != nil {
		s.logger.Error(internal.ComponentTransaction, "Initial transaction import failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.ProcessTransactions(); err != nil {
				s.logger.Error(internal.ComponentTransaction, "Transaction import failed: %v", err)
			}
		}
	}
}

// ProcessTransactions fetches and processes transactions from all configured sources
func (s *TransactionService) ProcessTransactions() error {
	// Process blockchain transactions
	if s.blockchainClient != nil {
		for chain, address := range s.config.Wallets {
			if err := s.processBlockchainTransactions(chain, address); err != nil {
				s.logger.Error(internal.ComponentTransaction, "Failed to process %s transactions: %v", chain, err)
			}
		}
	}

	// Process bank transactions
	if s.bankClient != nil {
		if err := s.processBankTransactions(); err != nil {
			s.logger.Error(internal.ComponentTransaction, "Failed to process bank transactions: %v", err)
		}
	}

	return nil
}

// processBlockchainTransactions processes transactions for a specific blockchain wallet
func (s *TransactionService) processBlockchainTransactions(chain, address string) error {
	s.logger.Info(internal.ComponentTransaction, "Processing %s transactions for %s", chain, address)

	// Fetch transactions
	transactions, err := s.blockchainClient.FetchTransactions(address)
	if err != nil {
		return fmt.Errorf("failed to fetch transactions: %w", err)
	}

	// Process each transaction
	for _, tx := range transactions {
		// Check if already imported
		imported, err := s.db.IsTransactionImported(tx.ID)
		if err != nil {
			s.logger.Error(internal.ComponentTransaction, "Failed to check transaction %s: %v", tx.ID, err)
			continue
		}
		if imported {
			continue
		}

		// Get currency ID from Firefly
		currencyID, err := s.fireflyClient.GetCurrencyID(chain)
		if err != nil {
			s.logger.Error(internal.ComponentTransaction, "Failed to get currency ID for %s: %v", chain, err)
			continue
		}

		// Create transaction in Firefly
		if err := s.fireflyClient.CreateTransaction(address, currencyID, tx); err != nil {
			s.logger.Error(internal.ComponentTransaction, "Failed to create transaction %s: %v", tx.ID, err)
			continue
		}

		// Mark as imported
		if err := s.db.MarkTransactionAsImported(tx.ID, map[string]string{
			"chain":   chain,
			"address": address,
		}); err != nil {
			s.logger.Error(internal.ComponentTransaction, "Failed to mark transaction %s as imported: %v", tx.ID, err)
		}
	}

	return nil
}

// processBankTransactions processes transactions from configured bank accounts
func (s *TransactionService) processBankTransactions() error {
	s.logger.Info(internal.ComponentTransaction, "Processing bank transactions")

	// Get last import time
	lastImport, err := s.db.GetLastImportTime("bank")
	if err != nil {
		s.logger.Warn(internal.ComponentTransaction, "Failed to get last import time: %v", err)
		lastImport = time.Now().AddDate(0, -1, 0) // Default to 1 month ago
	}

	// Format dates for bank API
	fromDate := lastImport.Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	// Fetch transactions with configured limit
	transactions, err := s.bankClient.FetchTransactions(s.config.BankAccounts[0].Limit, fromDate, toDate)
	if err != nil {
		return fmt.Errorf("failed to fetch bank transactions: %w", err)
	}

	// Process each transaction
	for _, tx := range transactions {
		// Check if already imported
		imported, err := s.db.IsTransactionImported(tx.ID)
		if err != nil {
			s.logger.Error(internal.ComponentTransaction, "Failed to check transaction %s: %v", tx.ID, err)
			continue
		}
		if imported {
			continue
		}

		// Get currency ID from Firefly
		currencyID, err := s.fireflyClient.GetCurrencyID(tx.Currency)
		if err != nil {
			s.logger.Error(internal.ComponentTransaction, "Failed to get currency ID for %s: %v", tx.Currency, err)
			continue
		}

		// Create transaction in Firefly
		if err := s.fireflyClient.CreateTransaction(s.bankClient.GetAccountName(), currencyID, tx); err != nil {
			s.logger.Error(internal.ComponentTransaction, "Failed to create transaction %s: %v", tx.ID, err)
			continue
		}

		// Mark as imported
		if err := s.db.MarkTransactionAsImported(tx.ID, map[string]string{
			"bank":     s.bankClient.GetName(),
			"account":  s.bankClient.GetAccountName(),
			"currency": tx.Currency,
		}); err != nil {
			s.logger.Error(internal.ComponentTransaction, "Failed to mark transaction %s as imported: %v", tx.ID, err)
		}
	}

	// Update last import time
	if err := s.db.SetLastImportTime("bank", time.Now()); err != nil {
		s.logger.Error(internal.ComponentTransaction, "Failed to update last import time: %v", err)
	}

	return nil
}

// parseInterval parses the interval string (e.g., "15m") into a duration
func (s *TransactionService) parseInterval(interval string) time.Duration {
	duration, err := time.ParseDuration(interval)
	if err != nil {
		s.logger.Warn(internal.ComponentTransaction, "Invalid interval %s, using 15m default", interval)
		return 15 * time.Minute
	}
	return duration
}
