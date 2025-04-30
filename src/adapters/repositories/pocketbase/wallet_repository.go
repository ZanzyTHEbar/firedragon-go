package pocketbase

import (
	"context"
	"fmt"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/domain/models"
	"github.com/ZanzyTHEbar/firedragon-go/domain/repositories"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// WalletRepository is a PocketBase implementation of the WalletRepository interface
type WalletRepository struct {
	app *pocketbase.PocketBase
}

// NewWalletRepository creates a new PocketBase wallet repository
func NewWalletRepository(app *pocketbase.PocketBase) *WalletRepository {
	return &WalletRepository{
		app: app,
	}
}

// FindByID finds a wallet by ID
func (r *WalletRepository) FindByID(ctx context.Context, id string) (*models.Wallet, error) {
	record, err := r.app.FindRecordById("wallets", id)
	if err != nil {
		return nil, fmt.Errorf("failed to find wallet: %w", err)
	}

	return r.mapRecordToWallet(record)
}

// FindAll finds all wallets with optional filters
func (r *WalletRepository) FindAll(ctx context.Context, filter repositories.WalletFilter) ([]*models.Wallet, error) {
	query := r.app.RecordQuery("wallets")

	// Apply filters
	if filter.Type != "" {
		query = query.AndWhere(dbx.HashExp{"type": string(filter.Type)})
	}

	if filter.Currency != "" {
		query = query.AndWhere(dbx.HashExp{"currency": filter.Currency})
	}

	if filter.NameLike != "" {
		query = query.AndWhere(dbx.NewExp("name LIKE {:name}", dbx.Params{"name": "%" + filter.NameLike + "%"}))
	}

	// Apply sorting
	if filter.SortBy != "" {
		direction := "ASC"
		if filter.SortOrder == "desc" {
			direction = "DESC"
		}
		query = query.OrderBy(fmt.Sprintf("%s %s", filter.SortBy, direction))
	} else {
		// Default sort by name ascending
		query = query.OrderBy("name ASC")
	}

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(int64(filter.Limit))
	}

	if filter.Offset > 0 {
		query = query.Offset(int64(filter.Offset))
	}

	// Execute query
	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, fmt.Errorf("failed to find wallets: %w", err)
	}

	// Convert records to domain models
	wallets := make([]*models.Wallet, 0, len(records))
	for _, record := range records {
		wallet, err := r.mapRecordToWallet(record)
		if err != nil {
			return nil, fmt.Errorf("failed to map record to wallet: %w", err)
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

// Create creates a new wallet
func (r *WalletRepository) Create(ctx context.Context, wallet *models.Wallet) error {
	record := r.mapWalletToRecord(wallet)

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	// Update the wallet ID from the saved record
	wallet.ID = record.Id

	return nil
}

// Update updates an existing wallet
func (r *WalletRepository) Update(ctx context.Context, wallet *models.Wallet) error {
	// Check if wallet exists
	record, err := r.app.FindRecordById("wallets", wallet.ID)
	if err != nil {
		return fmt.Errorf("failed to find wallet: %w", err)
	}

	// Update fields
	record = r.updateRecordFromWallet(record, wallet)

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}

	return nil
}

// Delete deletes a wallet by ID
func (r *WalletRepository) Delete(ctx context.Context, id string) error {
	record, err := r.app.FindRecordById("wallets", id) // Use r.app directly
	if err != nil {
		return fmt.Errorf("failed to find wallet: %w", err)
	}

	// Check for transactions associated with this wallet
	var txCount int64 // Use int64 for count
	// Select count(*) and use Row() to scan the result
	countQuery := r.app.RecordQuery("transactions").Select("count(*)").AndWhere(dbx.Or(dbx.HashExp{"wallet": id}, dbx.HashExp{"destination_wallet": id}))
	if err := countQuery.Row(&txCount); err != nil { // Use Row() to get the count
		return fmt.Errorf("failed to check for transactions using wallet: %w", err)
	}

	if txCount > 0 {
		return fmt.Errorf("wallet cannot be deleted because it has %d associated transactions", txCount)
	}

	if err := r.app.Delete(record); err != nil { // Use r.app directly
		return fmt.Errorf("failed to delete wallet: %w", err)
	}

	return nil
}

// UpdateBalance updates a wallet balance
func (r *WalletRepository) UpdateBalance(ctx context.Context, id string, amount float64) error {
	record, err := r.app.FindRecordById("wallets", id)
	if err != nil {
		return fmt.Errorf("failed to find wallet: %w", err)
	}

	currentBalance := record.GetFloat("balance")
	record.Set("balance", currentBalance+amount)

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	return nil
}

// FindByName finds a wallet by name (case-insensitive)
func (r *WalletRepository) FindByName(ctx context.Context, name string) (*models.Wallet, error) {
	record := &core.Record{}
	err := r.app.RecordQuery("wallets").
		AndWhere(dbx.NewExp("LOWER(name) = LOWER({:name})", dbx.Params{"name": name})).
		Limit(1).
		One(record)
	if err != nil {
		return nil, fmt.Errorf("failed to find wallet: %w", err)
	}

	return r.mapRecordToWallet(record)
}

// Helper methods for mapping between domain models and PocketBase records

func (r *WalletRepository) mapRecordToWallet(record *core.Record) (*models.Wallet, error) {
	wallet := &models.Wallet{
		ID:          record.Id,
		Name:        record.GetString("name"),
		Description: record.GetString("description"),
		Balance:     record.GetFloat("balance"),
		Currency:    record.GetString("currency"),
		Type:        models.WalletType(record.GetString("type")),
		CreatedAt:   record.GetDateTime("created").Time(),
		UpdatedAt:   record.GetDateTime("updated").Time(),
	}

	return wallet, nil
}

func (r *WalletRepository) mapWalletToRecord(wallet *models.Wallet) *core.Record {
	collection, _ := r.app.FindCollectionByNameOrId("wallets")
	record := core.NewRecord(collection)

	// Set basic fields
	record.Set("name", wallet.Name)
	record.Set("description", wallet.Description)
	record.Set("balance", wallet.Balance)
	record.Set("currency", wallet.Currency)
	record.Set("type", string(wallet.Type))

	// Set ID if specified
	if wallet.ID != "" {
		record.Id = wallet.ID
	}

	// Update timestamps if not set
	if wallet.CreatedAt.IsZero() {
		wallet.CreatedAt = time.Now()
	}
	wallet.UpdatedAt = time.Now()

	return record
}

func (r *WalletRepository) updateRecordFromWallet(record *core.Record, wallet *models.Wallet) *core.Record {
	// Update fields
	record.Set("name", wallet.Name)
	record.Set("description", wallet.Description)
	record.Set("balance", wallet.Balance)
	record.Set("currency", wallet.Currency)
	record.Set("type", string(wallet.Type))

	return record
}
