package repositories

import (
	"context"
)

// UnitOfWork represents a transactional unit of work
type UnitOfWork interface {
	// Begin starts a new transaction
	Begin(ctx context.Context) (context.Context, error)

	// Commit commits the current transaction
	Commit(ctx context.Context) error

	// Rollback rolls back the current transaction
	Rollback(ctx context.Context) error

	// RunInTransaction executes the given function in a transaction
	// and commits or rolls back automatically based on the function result
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// GetWalletRepository returns the wallet repository
	GetWalletRepository() WalletRepository

	// GetCategoryRepository returns the category repository
	GetCategoryRepository() CategoryRepository

	// GetTransactionRepository returns the transaction repository
	GetTransactionRepository() TransactionRepository
}
