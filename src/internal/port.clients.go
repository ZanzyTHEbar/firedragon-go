package internal

import (
	"time"
)

// BlockchainClient defines the interface for blockchain transaction adapters
type BlockchainClient interface {
	// FetchTransactions retrieves transactions for a specific wallet address
	FetchTransactions(address string) ([]Transaction, error)

	// GetBalance retrieves the current balance for a specific wallet address
	GetBalance(address string) (float64, error)

	// GetName returns the name of the blockchain (e.g., "ethereum", "solana", "sui")
	GetName() string
}

// BankAccountClient defines the interface for bank account adapters
type BankAccountClient interface {
	// FetchBalances retrieves all account balances
	FetchBalances() ([]Balance, error)

	// FetchTransactions retrieves transactions with customization options
	FetchTransactions(limit int, fromDate, toDate string) ([]Transaction, error)

	// GetName returns the name of the bank provider (e.g., "enable_banking")
	GetName() string

	// GetAccountName returns the account identifier
	GetAccountName() string
}

// FireflyClient defines the interface for Firefly III API interactions
type FireflyClient interface {
	// CreateTransaction creates a new transaction in Firefly III
	CreateTransaction(accountID, currencyID string, t Transaction) error

	// GetCurrencyID retrieves the Firefly III currency ID for a given account
	GetCurrencyID(accountID string) (string, error)

	// GetAccounts retrieves all accounts from Firefly III
	GetAccounts() (map[string]string, error)
}

// DatabaseClient defines the interface for database operations
type DatabaseClient interface {
	// IsTransactionImported checks if a transaction has already been imported
	IsTransactionImported(txID string) (bool, error)

	// MarkTransactionAsImported marks a transaction as imported
	MarkTransactionAsImported(txID string, metadata map[string]string) error

	// GetLastImportTime gets the timestamp of the last import operation
	GetLastImportTime(source string) (time.Time, error)

	// SetLastImportTime sets the timestamp of the last import operation
	SetLastImportTime(source string, timestamp time.Time) error

	// Close closes the database connection
	Close() error
}
