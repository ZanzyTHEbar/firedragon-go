package models

import (
	"time"

	"github.com/google/uuid"
)

// TransactionType defines the type of transaction
type TransactionType string

const (
	// TransactionTypeIncome represents an income transaction
	TransactionTypeIncome TransactionType = "income"
	
	// TransactionTypeExpense represents an expense transaction
	TransactionTypeExpense TransactionType = "expense"
	
	// TransactionTypeTransfer represents a transfer between wallets
	TransactionTypeTransfer TransactionType = "transfer"
)

// TransactionStatus defines the status of a transaction
type TransactionStatus string

const (
	// TransactionStatusPending represents a pending transaction
	TransactionStatusPending TransactionStatus = "pending"
	
	// TransactionStatusCompleted represents a completed transaction
	TransactionStatusCompleted TransactionStatus = "completed"
	
	// TransactionStatusFailed represents a failed transaction
	TransactionStatusFailed TransactionStatus = "failed"
)

// Transaction represents a financial transaction in the system
type Transaction struct {
	ID              string            `json:"id"`
	Amount          float64           `json:"amount"`
	Description     string            `json:"description"`
	Date            time.Time         `json:"date"`
	Type            TransactionType   `json:"type"`
	Status          TransactionStatus `json:"status"`
	CategoryID      string            `json:"categoryId"`
	WalletID        string            `json:"walletId"`
	DestWalletID    string            `json:"destWalletId,omitempty"`
	ExchangeRate    float64           `json:"exchangeRate,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// NewTransaction creates a new transaction with defaults
func NewTransaction(amount float64, description string, date time.Time, txType TransactionType, 
					categoryID, walletID string) *Transaction {
	return &Transaction{
		ID:          uuid.New().String(),
		Amount:      amount,
		Description: description,
		Date:        date,
		Type:        txType,
		Status:      TransactionStatusPending,
		CategoryID:  categoryID,
		WalletID:    walletID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// SetDestinationWallet sets the destination wallet for a transfer transaction
func (t *Transaction) SetDestinationWallet(destWalletID string, exchangeRate float64) error {
	if t.Type != TransactionTypeTransfer {
		return ErrNotTransferTransaction
	}
	
	if destWalletID == t.WalletID {
		return ErrSameWallet
	}
	
	t.DestWalletID = destWalletID
	t.ExchangeRate = exchangeRate
	t.UpdatedAt = time.Now()
	
	return nil
}

// Validate checks if the transaction is valid
func (t *Transaction) Validate() error {
	// Amount must be positive
	if t.Amount <= 0 {
		return ErrInvalidAmount
	}
	
	// Date cannot be in the future
	if t.Date.After(time.Now()) {
		return ErrFutureDate
	}
	
	// Must have a wallet
	if t.WalletID == "" {
		return ErrMissingWallet
	}
	
	// Must have a category
	if t.CategoryID == "" {
		return ErrMissingCategory
	}
	
	// For transfers, must have a destination wallet
	if t.Type == TransactionTypeTransfer {
		if t.DestWalletID == "" {
			return ErrMissingDestWallet
		}
		
		// Source and destination wallets must be different
		if t.WalletID == t.DestWalletID {
			return ErrSameWallet
		}
	}
	
	return nil
}

// MarkAsCompleted marks the transaction as completed
func (t *Transaction) MarkAsCompleted() {
	t.Status = TransactionStatusCompleted
	t.UpdatedAt = time.Now()
}

// MarkAsFailed marks the transaction as failed
func (t *Transaction) MarkAsFailed() {
	t.Status = TransactionStatusFailed
	t.UpdatedAt = time.Now()
} 