package models

import (
	"time"

	"github.com/google/uuid"
)

// BalanceInfo holds balance and currency information.
type BalanceInfo struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// WalletType defines the type of wallet
type WalletType string

const (
	// WalletTypeBank represents a bank account
	WalletTypeBank WalletType = "bank"
	
	// WalletTypeCrypto represents a cryptocurrency wallet
	WalletTypeCrypto WalletType = "crypto"
	
	// WalletTypeCash represents a cash wallet
	WalletTypeCash WalletType = "cash"
)

// Wallet represents a financial wallet in the system
type Wallet struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Balance     float64    `json:"balance"`
	Currency    string     `json:"currency"`
	Type        WalletType `json:"type"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// NewWallet creates a new wallet with defaults
func NewWallet(name, description string, currency string, walletType WalletType) *Wallet {
	return &Wallet{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Balance:     0,
		Currency:    currency,
		Type:        walletType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// Validate checks if the wallet is valid
func (w *Wallet) Validate() error {
	// Name is required
	if w.Name == "" {
		return ErrMissingWalletName
	}
	
	// Currency is required
	if w.Currency == "" {
		return ErrMissingCurrency
	}
	
	return nil
}

// UpdateBalance updates the wallet balance
func (w *Wallet) UpdateBalance(amount float64) {
	w.Balance += amount
	w.UpdatedAt = time.Now()
}

// HasSufficientBalance checks if the wallet has sufficient balance for a withdrawal
func (w *Wallet) HasSufficientBalance(amount float64) bool {
	return w.Balance >= amount
}

// ProcessIncome handles an income transaction
func (w *Wallet) ProcessIncome(amount float64) {
	w.UpdateBalance(amount)
}

// ProcessExpense handles an expense transaction
func (w *Wallet) ProcessExpense(amount float64) error {
	if !w.HasSufficientBalance(amount) {
		return ErrInsufficientBalance
	}
	
	w.UpdateBalance(-amount)
	return nil
}

// ProcessTransferOut handles an outgoing transfer transaction
func (w *Wallet) ProcessTransferOut(amount float64) error {
	return w.ProcessExpense(amount)
}

// ProcessTransferIn handles an incoming transfer transaction
func (w *Wallet) ProcessTransferIn(amount float64, exchangeRate float64) {
	// If exchange rate is provided, apply it
	if exchangeRate > 0 {
		amount *= exchangeRate
	}
	
	w.ProcessIncome(amount)
}
