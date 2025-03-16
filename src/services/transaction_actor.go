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
	stopChan          chan struct{}

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

	baseActor := NewBaseActor("transaction_service", logger)

	return &TransactionActor{
		BaseActor:         *baseActor,
		blockchainClients: blockchainClients,
		bankClients:       bankClients,
		fireflyClient:     fireflyClient,
		db:                db,
		config:            config,
		stopChan:          make(chan struct{}),
	}
}

// Receive handles incoming messages
func (a *TransactionActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case StartMsg:
		a.onStart()
	case StopMsg:
		a.onStop()
	case ProcessRequest:
		response := a.onProcess(msg)
		if ctx.Sender() != nil {
			ctx.Respond(response)
		}
	case StatusRequestMsg:
		a.onStatusRequest(ctx)
	}
}

func (a *TransactionActor) onStart() {
	a.logger.Info(internal.ComponentTransaction, "Starting transaction actor")
	
	// Start periodic processing if interval is configured
	if a.config.Interval != "" {
		duration, err := time.ParseDuration(a.config.Interval)
		if err != nil {
			a.logger.Error(internal.ComponentTransaction, "Invalid interval format: %v", err)
			return
		}

		// Schedule periodic imports
		go func() {
			ticker := time.NewTicker(duration)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					a.onProcess(ProcessRequest{})
				case <-a.stopChan:
					return
				}
			}
		}()
	}
}

func (a *TransactionActor) onStop() {
	a.logger.Info(internal.ComponentTransaction, "Stopping transaction actor")
	close(a.stopChan)
}

func (a *TransactionActor) onStatusRequest(ctx *actor.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	response := StatusResponseMsg{
		Status:     interfaces.ServiceStatusRunning,
		LastActive: a.lastRunTime,
		ErrorCount: a.errorCount,
		LastError:  a.lastError,
		CustomStats: map[string]interface{}{
			"imported_count": a.importedCount,
		},
	}

	if ctx.Sender() != nil {
		ctx.Respond(response)
	}
}

func (a *TransactionActor) onProcess(req ProcessRequest) ProcessResponse {
	start := time.Now()
	var totalImported int
	var allErrors []error

	// Process blockchain transactions if not bank-only
	if !req.BankOnly {
		imported, errors := a.processBlockchainTransactions()
		totalImported += imported
		allErrors = append(allErrors, errors...)
	}

	// Process bank transactions if not blockchain-only
	if !req.BlockchainOnly {
		imported, errors := a.processBankTransactions()
		totalImported += imported
		allErrors = append(allErrors, errors...)
	}

	// Update statistics
	a.mu.Lock()
	a.lastRunTime = start
	a.importedCount += totalImported
	if len(allErrors) > 0 {
		a.lastError = allErrors[0]
		a.errorCount += len(allErrors)
	}
	a.mu.Unlock()

	return ProcessResponse{
		ImportedCount: totalImported,
		Errors:       allErrors,
		CompletedAt:  time.Now(),
	}
}

// processBlockchainTransactions imports transactions from configured wallets
func (a *TransactionActor) processBlockchainTransactions() (int, []error) {
	var (
		g          errgroup.Group
		mu         sync.Mutex
		importCount int
		errors      []error
	)

	for chain, address := range a.config.Wallets {
		client, ok := a.blockchainClients[chain]
		if !ok {
			continue // Skip unsupported chains
		}

		currentClient := client // Capture loop variable
		currentAddress := address
		currentChain := chain

		g.Go(func() error {
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
				currencyID, err := a.fireflyClient.GetCurrencyID(tx.Currency)
				if err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to get currency ID for %s: %v", tx.Currency, err)
					continue
				}

				// Create transaction in Firefly
				if err := a.fireflyClient.CreateTransaction(currentAddress, currencyID, tx); err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to create transaction %s: %v", tx.ID, err)
					continue
				}

				// Mark as imported
				if err := a.db.MarkTransactionAsImported(tx.ID, map[string]string{
						"chain":    currentChain,
						"address": currentAddress,
						"currency": tx.Currency,
				}); err != nil {
					a.logger.Error(internal.ComponentTransaction, "Failed to mark transaction %s as imported: %v", tx.ID, err)
				}

				localCount++
			}

			mu.Lock()
			importCount += localCount
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		// Individual errors are already collected
		a.logger.Error(internal.ComponentTransaction, "Error processing blockchain transactions: %v", err)
	}

	return importCount, errors
}

// processBankTransactions imports transactions from configured bank accounts
func (a *TransactionActor) processBankTransactions() (int, []error) {
	var (
		g          errgroup.Group
		mu         sync.Mutex
		importCount int
		errors      []error
	)

	for _, client := range a.bankClients {
		currentClient := client // Capture loop variable

		g.Go(func() error {
			// Fetch transactions
			transactions, err := currentClient.FetchTransactions(100, "", "") // Use reasonable defaults
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

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		a.logger.Error(internal.ComponentTransaction, "Error processing bank transactions: %v", err)
	}

	return importCount, errors
}

// RunImportCycle processes all transactions from all sources
func (a *TransactionActor) RunImportCycle() (int, []error) {
	blockchainCount, blockchainErrors := a.processBlockchainTransactions()
	bankCount, bankErrors := a.processBankTransactions()

	return blockchainCount + bankCount, append(blockchainErrors, bankErrors...)
}
