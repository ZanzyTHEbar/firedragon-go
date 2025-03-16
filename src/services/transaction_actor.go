package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/anthdm/hollywood/actor"
	"golang.org/x/sync/errgroup"
)

// TransactionActor handles fetching and processing transactions from various sources
type TransactionActor struct {
	BaseActor
	blockchainClients map[string]interfaces.BlockchainClient
	bankClients       []interfaces.BankAccountClient
	fireflyClient     interfaces.FireflyClient
	db                interfaces.DatabaseClient
	config            *internal.Config

	// Last run statistics
	mu            sync.Mutex
	lastRunTime   time.Time
	lastError     error
	errorCount    int
	importedCount int
}

// ProcessRequest is a message requesting processing of transactions
type ProcessRequest struct {
	BlockchainOnly bool
	BankOnly       bool
}

// ProcessResponse is the response after processing transactions
type ProcessResponse struct {
	ImportedCount int
	Errors        []error
	CompletedAt   time.Time
}

// NewTransactionActor creates a new transaction actor
func NewTransactionActor(
	blockchainClients map[string]interfaces.BlockchainClient,
	bankClients []interfaces.BankAccountClient,
	fireflyClient interfaces.FireflyClient,
	db interfaces.DatabaseClient,
	logger *internal.Logger,
	config *internal.Config,
) *TransactionActor {
	return &TransactionActor{
		BaseActor:         NewBaseActor("transaction_service", logger),
		blockchainClients: blockchainClients,
		bankClients:       bankClients,
		fireflyClient:     fireflyClient,
		db:                db,
		config:            config,
	}
}

// Receive implements the actor.Receiver interface
func (a *TransactionActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case StartMsg:
		a.logger.Info(internal.ComponentTransaction, "Transaction actor started")

		// Schedule periodic runs if not already running
		if a.config.Interval != "" {
			interval, err := time.ParseDuration(a.config.Interval)
			if err != nil {
				a.logger.Warn(internal.ComponentTransaction, "Invalid interval %s, using 15m default", a.config.Interval)
				interval = 15 * time.Minute
			}

			ctx.Engine().SendWithInterval(ctx.Self(), interval, ProcessRequest{})
		}

	case StopMsg:
		a.logger.Info(internal.ComponentTransaction, "Transaction actor stopping")

	case ProcessRequest:
		a.logger.Info(internal.ComponentTransaction, "Processing transactions")

		var errors []error
		importCount := 0

		// Process transactions based on the request flags
		if !msg.BankOnly {
			count, errs := a.processBlockchainTransactions(ctx)
			importCount += count
			errors = append(errors, errs...)
		}

		if !msg.BlockchainOnly {
			count, errs := a.processBankTransactions(ctx)
			importCount += count
			errors = append(errors, errs...)
		}

		// Update statistics
		a.mu.Lock()
		a.lastRunTime = time.Now()
		a.importedCount = importCount
		if len(errors) > 0 {
			a.lastError = errors[0]
			a.errorCount += len(errors)
		}
		a.mu.Unlock()

		// Log results
		if len(errors) > 0 {
			a.logger.Warn(internal.ComponentTransaction, "Transaction processing completed with %d errors", len(errors))
		} else {
			a.logger.Info(internal.ComponentTransaction, "Transaction processing completed successfully")
		}
		a.logger.Info(internal.ComponentTransaction, "Imported %d transactions", importCount)

	case StatusRequestMsg:
		a.mu.Lock()
		stats := map[string]interface{}{
			"lastRunTime":   a.lastRunTime,
			"importedCount": a.importedCount,
		}
		a.mu.Unlock()

		ctx.Respond(StatusResponseMsg{
			Status:      ServiceStatusRunning,
			LastActive:  a.lastRunTime,
			ErrorCount:  a.errorCount,
			LastError:   a.lastError,
			CustomStats: stats,
		})
	}
}

// processBlockchainTransactions imports transactions from blockchain wallets
func (a *TransactionActor) processBlockchainTransactions(ctx actor.Context) (int, []error) {
	var (
		importCount int
		errors      []error
		g           errgroup.Group
		mu          sync.Mutex // Protects importCount and errors
	)

	for chain, address := range a.config.Wallets {
		client, ok := a.blockchainClients[chain]
		if !ok {
			a.logger.Warn(internal.ComponentTransaction, "No client found for %s", chain)
			continue
		}

		// Capture loop variables to avoid closure issues
		currentChain := chain
		currentAddress := address
		currentClient := client

		g.Go(func() error {
			a.logger.Info(internal.ComponentTransaction, "Processing %s transactions for %s", currentChain, currentAddress)

			// Fetch transactions
			transactions, err := currentClient.FetchTransactions(currentAddress)
			if err != nil {
				errMsg := fmt.Errorf("failed to fetch %s transactions: %w", currentChain, err)
				mu.Lock()
				errors = append(errors, errMsg)
				mu.Unlock()
				return errMsg
			}

			localCount := 0
			// Process each transaction
			for _, tx := range transactions {
				// Check if already imported
				imported, err := a.db.IsTransactionImported(tx.ID)
				if err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to check transaction %s: %v", tx.ID, err)
					continue
				}
				if imported {
					continue
				}

				// Get currency ID from Firefly
				currencyID, err := a.fireflyClient.GetCurrencyID(currentChain)
				if err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to get currency ID for %s: %v", currentChain, err)
					continue
				}

				// Create transaction in Firefly
				if err := a.fireflyClient.CreateTransaction(currentAddress, currencyID, tx); err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to create transaction %s: %v", tx.ID, err)
					continue
				}

				// Mark as imported
				if err := a.db.MarkTransactionAsImported(tx.ID, map[string]string{
					"chain":   currentChain,
					"address": currentAddress,
				}); err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to mark transaction %s as imported: %v", tx.ID, err)
				}

				localCount++
			}

			mu.Lock()
			importCount += localCount
			mu.Unlock()

			a.logger.Info(internal.ComponentTransaction, "Processed %d transactions for %s wallet", localCount, currentChain)
			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		a.logger.Error(internal.ComponentTransaction, "Error processing blockchain transactions: %v", err)
	}

	return importCount, errors
}

// processBankTransactions imports transactions from configured bank accounts
func (a *TransactionActor) processBankTransactions(ctx actor.Context) (int, []error) {
	if len(a.bankClients) == 0 {
		return 0, nil
	}

	var (
		importCount int
		errors      []error
		g           errgroup.Group
		mu          sync.Mutex // Protects importCount and errors
	)

	for i, client := range a.bankClients {
		// Find matching bank config
		var bankConfig *internal.BankAccountConfig
		for j := range a.config.BankAccounts {
			if a.config.BankAccounts[j].Name == client.GetAccountName() {
				bankConfig = &a.config.BankAccounts[j]
				break
			}
		}

		if bankConfig == nil {
			err := fmt.Errorf("bank account configuration not found for: %s", client.GetAccountName())
			errors = append(errors, err)
			continue
		}

		// Capture loop variables
		currentClient := client
		currentConfig := bankConfig
		currentIndex := i

		g.Go(func() error {
			a.logger.Info(internal.ComponentTransaction, "Processing bank transactions for %s (client %d)",
				currentClient.GetAccountName(), currentIndex)

			// Get last import time
			lastImport, err := a.db.GetLastImportTime("bank:" + currentClient.GetAccountName())
			if err != nil {
				a.logger.Warn(internal.ComponentTransaction, "Failed to get last import time: %v", err)
				lastImport = time.Now().AddDate(0, -1, 0) // Default to 1 month ago
			}

			// Format dates for bank API
			fromDate := lastImport.Format("2006-01-02")
			toDate := time.Now().Format("2006-01-02")

			// Fetch transactions with configured limit
			transactions, err := currentClient.FetchTransactions(currentConfig.Limit, fromDate, toDate)
			if err != nil {
				errMsg := fmt.Errorf("failed to fetch bank transactions: %w", err)
				mu.Lock()
				errors = append(errors, errMsg)
				mu.Unlock()
				return errMsg
			}

			localCount := 0
			// Process each transaction
			for _, tx := range transactions {
				// Check if already imported
				imported, err := a.db.IsTransactionImported(tx.ID)
				if err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to check transaction %s: %v", tx.ID, err)
					continue
				}
				if imported {
					continue
				}

				// Get currency ID from Firefly
				currencyID, err := a.fireflyClient.GetCurrencyID(tx.Currency)
				if err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to get currency ID for %s: %v", tx.Currency, err)
					continue
				}

				// Create transaction in Firefly
				if err := a.fireflyClient.CreateTransaction(currentClient.GetAccountName(), currencyID, tx); err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to create transaction %s: %v", tx.ID, err)
					continue
				}

				// Mark as imported
				if err := a.db.MarkTransactionAsImported(tx.ID, map[string]string{
					"bank":     currentClient.GetName(),
					"account":  currentClient.GetAccountName(),
					"currency": tx.Currency,
				}); err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to mark transaction %s as imported: %v", tx.ID, err)
				}

				localCount++
			}

			mu.Lock()
			importCount += localCount
			mu.Unlock()

			// Update last import time
			if localCount > 0 {
				if err := a.db.SetLastImportTime("bank:"+currentClient.GetAccountName(), time.Now()); err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to update last import time: %v", err)
				}
			}

			a.logger.Info(internal.ComponentTransaction, "Processed %d transactions for bank account %s",
				localCount, currentClient.GetAccountName())
			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		a.logger.Error(internal.ComponentTransaction, "Error processing bank transactions: %v", err)
	}

	return importCount, errors
}

// RunImportCycle processes all transactions from all sources
func (a *TransactionActor) RunImportCycle() (int, []error) {
	blockchainCount, blockchainErrors := a.processBlockchainTransactions(nil)
	bankCount, bankErrors := a.processBankTransactions(nil)

	return blockchainCount + bankCount, append(blockchainErrors, bankErrors...)
}
