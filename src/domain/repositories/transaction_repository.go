package repositories

import (
	"context"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/domain/models"
)

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	// FindByID finds a transaction by ID
	FindByID(ctx context.Context, id string) (*models.Transaction, error)

	// FindAll finds all transactions with optional filters
	FindAll(ctx context.Context, filter TransactionFilter) ([]*models.Transaction, error)

	// Create creates a new transaction
	Create(ctx context.Context, transaction *models.Transaction) error

	// Update updates an existing transaction
	Update(ctx context.Context, transaction *models.Transaction) error

	// Delete deletes a transaction by ID
	Delete(ctx context.Context, id string) error

	// FindDuplicates finds potential duplicate transactions
	FindDuplicates(ctx context.Context, transaction *models.Transaction, timeWindow time.Duration) ([]*models.Transaction, error)
}

// TransactionFilter defines filters for finding transactions
type TransactionFilter struct {
	WalletID     string
	CategoryID   string
	Type         models.TransactionType
	DateFrom     time.Time
	DateTo       time.Time
	AmountMin    float64
	AmountMax    float64
	Description  string
	Status       models.TransactionStatus
	Limit        int
	Offset       int
	SortBy       string
	SortOrder    string
} 