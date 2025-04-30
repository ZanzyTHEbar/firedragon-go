package interfaces

import (
"time"

"github.com/ZanzyTHEbar/firedragon-go/domain/models"
)

// ErrorType represents different types of client errors
type ErrorType string

const (
	ErrorTypeNetwork  ErrorType = "network"
	ErrorTypeAuth     ErrorType = "auth"
	ErrorTypeInvalid  ErrorType = "invalid"
	ErrorTypeNotFound ErrorType = "not_found"
)

// ClientError represents an error from a client
type ClientError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *ClientError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// NewClientError creates a new client error
func NewClientError(errorType ErrorType, message string, err error) error {
	return &ClientError{
		Type:    errorType,
		Message: message,
		Err:     err,
	}
}

// BlockchainClient defines the interface for blockchain clients
type BlockchainClient interface {
// FetchTransactions retrieves transactions for a wallet address
FetchTransactions(address string) ([]models.Transaction, error)

// GetBalance gets the current balance for a wallet address
GetBalance(address string) (models.BalanceInfo, error)

// GetChainType returns the blockchain type (e.g., "ethereum", "solana")
GetChainType() string

	// IsValidAddress validates a wallet address format
	IsValidAddress(address string) bool
}

// BankClient defines the interface for banking clients
type BankClient interface {
// FetchTransactions retrieves transactions for a bank account
FetchTransactions(accountID string) ([]models.Transaction, error)

// GetBalance gets the current balance for a bank account
GetBalance(accountID string) (models.BalanceInfo, error)

// GetProviderType returns the bank provider type (e.g., "enable")
GetProviderType() string

	// ValidateCredentials validates the client's credentials
	ValidateCredentials() error

	// RefreshToken refreshes the OAuth token if needed
	RefreshToken() error
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

	// SearchSimilarTransactions finds transactions with similar metadata
	SearchSimilarTransactions(metadata map[string]string, limit int) ([]string, error)

	// Close closes the database connection
	Close() error
}

// MetricsClient defines the interface for metrics collection
type MetricsClient interface {
	// RecordImport records a transaction import event
	RecordImport(source, status string)

	// RecordError records an error event
	RecordError(source, errorType string)

	// RecordLatency records operation latency
	RecordLatency(operation string, duration time.Duration)

	// GetMetrics returns current metrics
	GetMetrics() map[string]interface{}
}
