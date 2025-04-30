package repositories

import (
	"context"

	"github.com/ZanzyTHEbar/firedragon-go/domain/models"
)

// WalletRepository defines the interface for wallet data access
type WalletRepository interface {
	// FindByID finds a wallet by ID
	FindByID(ctx context.Context, id string) (*models.Wallet, error)

	// FindAll finds all wallets with optional filters
	FindAll(ctx context.Context, filter WalletFilter) ([]*models.Wallet, error)

	// Create creates a new wallet
	Create(ctx context.Context, wallet *models.Wallet) error

	// Update updates an existing wallet
	Update(ctx context.Context, wallet *models.Wallet) error

	// Delete deletes a wallet by ID
	Delete(ctx context.Context, id string) error

	// UpdateBalance updates a wallet balance
	UpdateBalance(ctx context.Context, id string, amount float64) error
}

// WalletFilter defines filters for finding wallets
type WalletFilter struct {
	Type       models.WalletType
	Currency   string
	NameLike   string
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
} 