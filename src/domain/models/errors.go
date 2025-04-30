package models

import (
	"errors"
)

// Domain error types
var (
	// Transaction errors
	// ErrInvalidAmount is returned when a transaction amount is invalid
	ErrInvalidAmount = errors.New("transaction amount must be greater than 0")
	
	// ErrFutureDate is returned when a transaction date is in the future
	ErrFutureDate = errors.New("transaction date cannot be in the future")
	
	// ErrMissingWallet is returned when a transaction has no wallet
	ErrMissingWallet = errors.New("transaction must have a wallet")
	
	// ErrMissingCategory is returned when a transaction has no category
	ErrMissingCategory = errors.New("transaction must have a category")
	
	// ErrMissingDestWallet is returned when a transfer transaction has no destination wallet
	ErrMissingDestWallet = errors.New("transfer transaction must have a destination wallet")
	
	// ErrSameWallet is returned when a transfer transaction has the same source and destination wallet
	ErrSameWallet = errors.New("transfer transaction cannot have the same source and destination wallet")
	
	// ErrNotTransferTransaction is returned when trying to set a destination wallet on a non-transfer transaction
	ErrNotTransferTransaction = errors.New("destination wallet can only be set on transfer transactions")
	
	// ErrInsufficientBalance is returned when a wallet has insufficient balance for a transaction
	ErrInsufficientBalance = errors.New("wallet has insufficient balance for this transaction")
	
	// ErrCategoryTypeMismatch is returned when a transaction type doesn't match the category type
	ErrCategoryTypeMismatch = errors.New("transaction type doesn't match category type")
	
	// ErrDuplicateTransaction is returned when a duplicate transaction is detected
	ErrDuplicateTransaction = errors.New("duplicate transaction detected")
	
	// ErrInvalidExchangeRate is returned when a cross-currency transfer has an invalid exchange rate
	ErrInvalidExchangeRate = errors.New("cross-currency transfer must have a valid exchange rate")

	// Wallet errors
	// ErrMissingWalletName is returned when a wallet has no name
	ErrMissingWalletName = errors.New("wallet must have a name")
	
	// ErrMissingCurrency is returned when a wallet has no currency
	ErrMissingCurrency = errors.New("wallet must have a currency")
	
	// ErrWalletNotFound is returned when a wallet is not found
	ErrWalletNotFound = errors.New("wallet not found")
	
	// ErrInvalidCurrency is returned when a wallet has an invalid currency
	ErrInvalidCurrency = errors.New("invalid currency code")

	// Category errors
	// ErrMissingCategoryName is returned when a category has no name
	ErrMissingCategoryName = errors.New("category must have a name")
	
	// ErrInvalidCategoryType is returned when a category has an invalid type
	ErrInvalidCategoryType = errors.New("invalid category type")
	
	// ErrCategoryNotFound is returned when a category is not found
	ErrCategoryNotFound = errors.New("category not found")
	
	// ErrSystemCategoryCannotBeDeleted is returned when attempting to delete a system category
	ErrSystemCategoryCannotBeDeleted = errors.New("system categories cannot be deleted")
) 