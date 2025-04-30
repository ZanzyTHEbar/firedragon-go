package usecases

import (
	"context" // Add context import
	"fmt"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/domain/models"
	"github.com/ZanzyTHEbar/firedragon-go/domain/repositories"
	"github.com/ZanzyTHEbar/firedragon-go/internal" // For logging component type
)

// TransactionService encapsulates business logic related to transactions.
type TransactionService struct {
	walletRepo      repositories.WalletRepository
	categoryRepo    repositories.CategoryRepository
	transactionRepo repositories.TransactionRepository
	// Add other dependencies like a UnitOfWork or TxManager if needed
}

// NewTransactionService creates a new TransactionService.
func NewTransactionService(
	walletRepo repositories.WalletRepository,
	categoryRepo repositories.CategoryRepository,
	transactionRepo repositories.TransactionRepository,
) *TransactionService {
	return &TransactionService{
		walletRepo:      walletRepo,
		categoryRepo:    categoryRepo,
		transactionRepo: transactionRepo,
	}
}

// CreateTransactionInput defines the input for creating a transaction.
// Using specific input struct allows for better control over required fields.
type CreateTransactionInput struct {
	Amount       float64
	Description  string
	Date         time.Time
	Type         models.TransactionType
	CategoryID   string
	WalletID     string
	DestWalletID string   // Optional: for transfers
	ExchangeRate float64  // Optional: for transfers
	Tags         []string // Optional
}

// CreateTransaction handles the creation and processing of a new transaction.
// It performs validation, updates balances, and saves the transaction.
func (s *TransactionService) CreateTransaction(input CreateTransactionInput) (*models.Transaction, error) {
	logger := internal.GetLogger().With().Str("usecase", "CreateTransaction").Logger()
	logger.Info().Interface("input", input).Msg("Starting transaction creation")

	// --- 1. Validation ---
	logger.Debug().Msg("Validating input")

	// Basic input validation
	if input.Amount <= 0 {
		return nil, fmt.Errorf("invalid amount: %w", models.ErrInvalidAmount)
	}
	if input.Date.IsZero() || input.Date.After(time.Now()) {
		return nil, fmt.Errorf("invalid date: %w", models.ErrFutureDate)
	}
	if input.WalletID == "" {
		return nil, fmt.Errorf("missing wallet ID: %w", models.ErrMissingWallet)
	}
	if input.CategoryID == "" {
		return nil, fmt.Errorf("missing category ID: %w", models.ErrMissingCategory)
	}

	// Fetch related entities using repositories
	ctx := context.Background() // Use background context for now
	logger.Debug().Str("walletID", input.WalletID).Msg("Fetching source wallet")
	sourceWallet, err := s.walletRepo.FindByID(ctx, input.WalletID)
	if err != nil {
		logger.Error().Err(err).Str("walletID", input.WalletID).Msg("Failed to fetch source wallet")
		return nil, fmt.Errorf("failed to get source wallet: %w", err)
	}

	logger.Debug().Str("categoryID", input.CategoryID).Msg("Fetching category")
	category, err := s.categoryRepo.FindByID(ctx, input.CategoryID)
	if err != nil {
		logger.Error().Err(err).Str("categoryID", input.CategoryID).Msg("Failed to fetch category")
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	// Validate category type matches transaction type
	if !category.MatchesTransactionType(input.Type) {
		err := fmt.Errorf("category type '%s' does not match transaction type '%s': %w",
			category.Type, input.Type, models.ErrInvalidCategoryType)
		logger.Warn().Err(err).Msg("Category type mismatch")
		return nil, err
	}

	var destWallet *models.Wallet // Declare outside switch for later use

	// Type-specific validation
	switch input.Type {
	case models.TransactionTypeExpense:
		if !sourceWallet.HasSufficientBalance(input.Amount) {
			logger.Warn().Float64("balance", sourceWallet.Balance).Float64("amount", input.Amount).Msg("Insufficient balance for expense")
			return nil, fmt.Errorf("insufficient balance in source wallet: %w", models.ErrInsufficientBalance)
		}
	case models.TransactionTypeTransfer:
		if input.DestWalletID == "" {
			return nil, fmt.Errorf("missing destination wallet ID for transfer: %w", models.ErrMissingDestWallet)
		}
		if input.DestWalletID == input.WalletID {
			return nil, fmt.Errorf("source and destination wallets cannot be the same: %w", models.ErrSameWallet)
		}

		logger.Debug().Str("destWalletID", input.DestWalletID).Msg("Fetching destination wallet")
		destWallet, err = s.walletRepo.FindByID(ctx, input.DestWalletID)
		if err != nil {
			logger.Error().Err(err).Str("destWalletID", input.DestWalletID).Msg("Failed to fetch destination wallet")
			return nil, fmt.Errorf("failed to get destination wallet: %w", err)
		}

		if !sourceWallet.HasSufficientBalance(input.Amount) {
			logger.Warn().Float64("balance", sourceWallet.Balance).Float64("amount", input.Amount).Msg("Insufficient balance for transfer")
			return nil, fmt.Errorf("insufficient balance in source wallet: %w", models.ErrInsufficientBalance)
		}

		// Validate exchange rate if currencies differ
		if sourceWallet.Currency != destWallet.Currency && input.ExchangeRate <= 0 {
			err := fmt.Errorf("exchange rate is required for cross-currency transfer: %w", models.ErrInvalidExchangeRate)
			logger.Warn().Err(err).
				Str("sourceCurrency", sourceWallet.Currency).
				Str("destCurrency", destWallet.Currency).
				Msg("Missing exchange rate")
			return nil, err
		}
	}

	// --- 2. Duplicate Check (Placeholder - refine logic) ---
// TODO: Implement duplicate check logic using transactionRepo
// Define a time window for duplicate checks (e.g., 24 hours)
duplicateCheckWindow := 24 * time.Hour
potentialDuplicates, err := s.transactionRepo.FindDuplicates(ctx, &models.Transaction{
Amount:       input.Amount,
Date:         input.Date,
Type:         input.Type,
CategoryID:   input.CategoryID,
WalletID:     input.WalletID,
DestWalletID: input.DestWalletID, // Include DestWalletID for transfers
}, duplicateCheckWindow)
if err != nil {
// Log error but potentially continue? Or return error? Deciding to log and continue for now.
logger.Error().Err(err).Msg("Failed to check for duplicate transactions")
} else if len(potentialDuplicates) > 0 {
logger.Warn().Int("count", len(potentialDuplicates)).Msg("Potential duplicate transaction(s) detected")
// Return an error to prevent duplicate creation
return nil, fmt.Errorf("potential duplicate transaction detected (found %d similar)", len(potentialDuplicates))
} else {
logger.Debug().Msg("No potential duplicates found")
}

// --- 3. Create Transaction Entity ---
logger.Debug().Msg("Creating transaction entity")
tx := models.NewTransaction(
		input.Amount,
		input.Description,
		input.Date,
		input.Type,
		input.CategoryID,
		input.WalletID,
	)
	tx.Tags = input.Tags // Assign optional tags

	// Set transfer-specific fields
	if input.Type == models.TransactionTypeTransfer {
		if err := tx.SetDestinationWallet(input.DestWalletID, input.ExchangeRate); err != nil {
			// This should ideally be caught by earlier validation, but double-check
			logger.Error().Err(err).Msg("Failed to set destination wallet on transaction model")
			return nil, fmt.Errorf("internal error setting destination wallet: %w", err)
		}
	}

	// Final validation on the created model itself
	if err := tx.Validate(); err != nil {
		logger.Error().Err(err).Msg("Transaction model validation failed")
		return nil, fmt.Errorf("transaction model validation failed: %w", err)
	}

	// --- 4. Process Balance Updates (within a transaction/UoW if possible) ---
	// TODO: Wrap this section in a Unit of Work / DB transaction if the repo supports it.
	logger.Debug().Msg("Processing balance updates")
	switch tx.Type {
	case models.TransactionTypeIncome:
		sourceWallet.ProcessIncome(tx.Amount)
	case models.TransactionTypeExpense:
		if err := sourceWallet.ProcessExpense(tx.Amount); err != nil {
			// Should be caught by validation, but handle defensively
			logger.Error().Err(err).Msg("Error processing expense on source wallet")
			return nil, fmt.Errorf("failed to process expense: %w", err)
		}
	case models.TransactionTypeTransfer:
		if err := sourceWallet.ProcessTransferOut(tx.Amount); err != nil {
			// Should be caught by validation, but handle defensively
			logger.Error().Err(err).Msg("Error processing transfer out on source wallet")
			return nil, fmt.Errorf("failed to process transfer out: %w", err)
		}
		// Process transfer in for destination wallet (must exist from validation step)
		destWallet.ProcessTransferIn(tx.Amount, tx.ExchangeRate)

// Update destination wallet changes
logger.Debug().Str("destWalletID", destWallet.ID).Msg("Updating destination wallet")
if err := s.walletRepo.Update(ctx, destWallet); err != nil {
logger.Error().Err(err).Str("destWalletID", destWallet.ID).Msg("Failed to update destination wallet")
// TODO: Rollback transaction if applicable
return nil, fmt.Errorf("failed to update destination wallet: %w", err)
}
}

// Update source wallet changes
logger.Debug().Str("sourceWalletID", sourceWallet.ID).Msg("Updating source wallet")
if err := s.walletRepo.Update(ctx, sourceWallet); err != nil {
logger.Error().Err(err).Str("sourceWalletID", sourceWallet.ID).Msg("Failed to update source wallet")
// TODO: Rollback transaction if applicable
return nil, fmt.Errorf("failed to update source wallet: %w", err)
}

// --- 5. Create Transaction Record ---
tx.MarkAsCompleted() // Mark as completed after successful processing
logger.Debug().Str("transactionID", tx.ID).Msg("Creating transaction record")
if err := s.transactionRepo.Create(ctx, tx); err != nil {
logger.Error().Err(err).Str("transactionID", tx.ID).Msg("Failed to create transaction record")
// TODO: Rollback transaction if applicable
// Consider marking wallet balances back? Complex without UoW.
return nil, fmt.Errorf("failed to create transaction record: %w", err)
	}

	logger.Info().Str("transactionID", tx.ID).Msg("Transaction created successfully")
	return tx, nil
}

// TODO: Add methods for UpdateTransaction, DeleteTransaction, GetTransactionByID etc.
// These would involve similar steps: fetch, validate, process (including reversals), save.
