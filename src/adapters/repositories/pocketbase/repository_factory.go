package pocketbase

import (
	"github.com/ZanzyTHEbar/firedragon-go/domain/repositories"
	"github.com/pocketbase/pocketbase"
)

// RepositoryFactory creates PocketBase-backed repositories
type RepositoryFactory struct {
	app *pocketbase.PocketBase
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(app *pocketbase.PocketBase) *RepositoryFactory {
	return &RepositoryFactory{
		app: app,
	}
}

// CreateTransactionRepository creates a new transaction repository
func (f *RepositoryFactory) CreateTransactionRepository() repositories.TransactionRepository {
	return NewTransactionRepository(f.app)
}

// CreateWalletRepository creates a new wallet repository
func (f *RepositoryFactory) CreateWalletRepository() repositories.WalletRepository {
	return NewWalletRepository(f.app)
}

// CreateCategoryRepository creates a new category repository
func (f *RepositoryFactory) CreateCategoryRepository() repositories.CategoryRepository {
	return NewCategoryRepository(f.app)
}

// CreateUnitOfWork creates a new unit of work
func (f *RepositoryFactory) CreateUnitOfWork() repositories.UnitOfWork {
	return NewPocketBaseUnitOfWork(f.app)
}
