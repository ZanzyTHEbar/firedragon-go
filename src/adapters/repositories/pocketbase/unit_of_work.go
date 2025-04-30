package pocketbase

import (
	"context"
	"fmt"

	"github.com/ZanzyTHEbar/firedragon-go/domain/repositories"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// PocketBaseUnitOfWork implements the UnitOfWork interface for PocketBase
type PocketBaseUnitOfWork struct {
	app             *pocketbase.PocketBase
	walletRepo      repositories.WalletRepository
	categoryRepo    repositories.CategoryRepository
	transactionRepo repositories.TransactionRepository
}

// NewPocketBaseUnitOfWork creates a new PocketBase unit of work
func NewPocketBaseUnitOfWork(app *pocketbase.PocketBase) *PocketBaseUnitOfWork {
	return &PocketBaseUnitOfWork{
		app:             app,
		walletRepo:      NewWalletRepository(app),
		categoryRepo:    NewCategoryRepository(app),
		transactionRepo: NewTransactionRepository(app),
	}
}

// Begin starts a new transaction
// In PocketBase, we don't actually start a transaction here, as PocketBase's RunInTransaction
// handles this for us. This method is primarily for interface compatibility.
func (uow *PocketBaseUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	// PocketBase doesn't have a way to start a transaction and pass it around via context
	// So we'll just return the context as is
	return ctx, nil
}

// Commit commits the current transaction
// In PocketBase, this is a no-op as we use RunInTransaction which handles commits automatically
func (uow *PocketBaseUnitOfWork) Commit(ctx context.Context) error {
	// No explicit commit needed when using RunInTransaction
	return nil
}

// Rollback rolls back the current transaction
// In PocketBase, this is a no-op as we use RunInTransaction which handles rollbacks automatically
func (uow *PocketBaseUnitOfWork) Rollback(ctx context.Context) error {
	// No explicit rollback needed when using RunInTransaction
	return nil
}

// RunInTransaction executes the given function in a transaction
func (uow *PocketBaseUnitOfWork) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return uow.app.RunInTransaction(func(txApp core.App) error {
		// Create a new context with the transaction app
		txCtx := context.WithValue(ctx, "txApp", txApp)

		// Execute the function
		if err := fn(txCtx); err != nil {
			// Function returned an error, transaction will be rolled back automatically
			return fmt.Errorf("transaction failed: %w", err)
		}

		// Function succeeded, transaction will be committed automatically
		return nil
	})
}

// GetWalletRepository returns the wallet repository
func (uow *PocketBaseUnitOfWork) GetWalletRepository() repositories.WalletRepository {
	return uow.walletRepo
}

// GetCategoryRepository returns the category repository
func (uow *PocketBaseUnitOfWork) GetCategoryRepository() repositories.CategoryRepository {
	return uow.categoryRepo
}

// GetTransactionRepository returns the transaction repository
func (uow *PocketBaseUnitOfWork) GetTransactionRepository() repositories.TransactionRepository {
	return uow.transactionRepo
}
