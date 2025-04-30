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
	"github.com/pocketbase/pocketbase/daos"
)

// TransactionRepository is a PocketBase implementation of the TransactionRepository interface
type TransactionRepository struct {
	app *pocketbase.PocketBase
	dao *daos.Dao
}

// NewTransactionRepository creates a new PocketBase transaction repository
func NewTransactionRepository(app *pocketbase.PocketBase) *TransactionRepository {
	return &TransactionRepository{
		app: app,
		dao: app.Dao(),
	}
}

// FindByID finds a transaction by ID
func (r *TransactionRepository) FindByID(ctx context.Context, id string) (*models.Transaction, error) {
	record, err := r.dao.FindRecordById("transactions", id)
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}

	return r.mapRecordToTransaction(record)
}

// FindAll finds all transactions with optional filters
func (r *TransactionRepository) FindAll(ctx context.Context, filter repositories.TransactionFilter) ([]*models.Transaction, error) {
	query := r.dao.RecordQuery("transactions")

	// Apply filters
	if filter.WalletID != "" {
		query = query.AndWhere(dbx.HashExp{"wallet": filter.WalletID})
	}

	if filter.CategoryID != "" {
		query = query.AndWhere(dbx.HashExp{"category": filter.CategoryID})
	}

	if filter.Type != "" {
		query = query.AndWhere(dbx.HashExp{"type": string(filter.Type)})
	}

	if filter.Description != "" {
		query = query.AndWhere(dbx.NewExp("description LIKE {:desc}", dbx.Params{"desc": "%" + filter.Description + "%"}))
	}

	if !filter.DateFrom.IsZero() {
		query = query.AndWhere(dbx.NewExp("date >= {:date_from}", dbx.Params{"date_from": filter.DateFrom}))
	}

	if !filter.DateTo.IsZero() {
		query = query.AndWhere(dbx.NewExp("date <= {:date_to}", dbx.Params{"date_to": filter.DateTo}))
	}

	if filter.AmountMin > 0 {
		query = query.AndWhere(dbx.NewExp("amount >= {:amount_min}", dbx.Params{"amount_min": filter.AmountMin}))
	}

	if filter.AmountMax > 0 {
		query = query.AndWhere(dbx.NewExp("amount <= {:amount_max}", dbx.Params{"amount_max": filter.AmountMax}))
	}

	if filter.Status != "" {
		query = query.AndWhere(dbx.HashExp{"status": string(filter.Status)})
	}

	// Apply sorting
	if filter.SortBy != "" {
		direction := "ASC"
		if filter.SortOrder == "desc" {
			direction = "DESC"
		}
		query = query.OrderBy(fmt.Sprintf("%s %s", filter.SortBy, direction))
	} else {
		// Default sort by date descending
		query = query.OrderBy("date DESC")
	}

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	// Execute query
	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, fmt.Errorf("failed to find transactions: %w", err)
	}

	// Convert records to domain models
	transactions := make([]*models.Transaction, 0, len(records))
	for _, record := range records {
		transaction, err := r.mapRecordToTransaction(record)
		if err != nil {
			return nil, fmt.Errorf("failed to map record to transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// Create creates a new transaction
func (r *TransactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	record := r.mapTransactionToRecord(transaction)

	if err := r.dao.Save(record); err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// Update updates an existing transaction
func (r *TransactionRepository) Update(ctx context.Context, transaction *models.Transaction) error {
	// Check if transaction exists
	record, err := r.dao.FindRecordById("transactions", transaction.ID)
	if err != nil {
		return fmt.Errorf("failed to find transaction: %w", err)
	}

	// Update fields
	record = r.updateRecordFromTransaction(record, transaction)

	if err := r.dao.Save(record); err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

// Delete deletes a transaction by ID
func (r *TransactionRepository) Delete(ctx context.Context, id string) error {
	record, err := r.dao.FindRecordById("transactions", id)
	if err != nil {
		return fmt.Errorf("failed to find transaction: %w", err)
	}

	if err := r.dao.Delete(record); err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	return nil
}

// FindDuplicates finds potential duplicate transactions
func (r *TransactionRepository) FindDuplicates(ctx context.Context, transaction *models.Transaction, timeWindow time.Duration) ([]*models.Transaction, error) {
	// Calculate time range for duplicate check
	startTime := transaction.Date.Add(-timeWindow / 2)
	endTime := transaction.Date.Add(timeWindow / 2)

	// Build query for potential duplicates
	query := r.dao.RecordQuery("transactions").
		AndWhere(dbx.HashExp{"wallet": transaction.WalletID}).
		AndWhere(dbx.NewExp("ABS(amount - {:amount}) < 0.01", dbx.Params{"amount": transaction.Amount})).
		AndWhere(dbx.NewExp("date >= {:start_date}", dbx.Params{"start_date": startTime})).
		AndWhere(dbx.NewExp("date <= {:end_date}", dbx.Params{"end_date": endTime})).
		AndWhere(dbx.HashExp{"type": string(transaction.Type)})

	// For transfers, also check destination wallet
	if transaction.Type == models.TransactionTypeTransfer && transaction.DestWalletID != "" {
		query = query.AndWhere(dbx.HashExp{"destination_wallet": transaction.DestWalletID})
	}

	// Don't include the transaction itself if it has an ID
	if transaction.ID != "" {
		query = query.AndWhere(dbx.NewExp("id <> {:id}", dbx.Params{"id": transaction.ID}))
	}

	// Execute query
	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, fmt.Errorf("failed to find duplicate transactions: %w", err)
	}

	// Convert records to domain models
	duplicates := make([]*models.Transaction, 0, len(records))
	for _, record := range records {
		duplicate, err := r.mapRecordToTransaction(record)
		if err != nil {
			return nil, fmt.Errorf("failed to map record to transaction: %w", err)
		}
		duplicates = append(duplicates, duplicate)
	}

	return duplicates, nil
}

// Helper methods for mapping between domain models and PocketBase records

func (r *TransactionRepository) mapRecordToTransaction(record *core.Record) (*models.Transaction, error) {
	// Create transaction with basic fields
	tx := &models.Transaction{
		ID:          record.Id,
		Amount:      record.GetFloat("amount"),
		Description: record.GetString("description"),
		Date:        record.GetDateTime("date").Time(),
		Type:        models.TransactionType(record.GetString("type")),
		Status:      models.TransactionStatus(record.GetString("status")),
		CategoryID:  record.GetString("category"),
		WalletID:    record.GetString("wallet"),
		CreatedAt:   record.GetDateTime("created").Time(),
		UpdatedAt:   record.GetDateTime("updated").Time(),
	}

	// Handle transfer-specific fields
	if tx.Type == models.TransactionTypeTransfer {
		tx.DestWalletID = record.GetString("destination_wallet")
		tx.ExchangeRate = record.GetFloat("exchange_rate")
	}

	// Handle tags if present
	if record.Has("tags") {
		tags := record.Get("tags")
		if tagsArray, ok := tags.([]interface{}); ok {
			tx.Tags = make([]string, 0, len(tagsArray))
			for _, tag := range tagsArray {
				if tagStr, ok := tag.(string); ok {
					tx.Tags = append(tx.Tags, tagStr)
				}
			}
		}
	}

	return tx, nil
}

func (r *TransactionRepository) mapTransactionToRecord(transaction *models.Transaction) *core.Record {
	collection, _ := r.dao.FindCollectionByNameOrId("transactions")
	record := core.NewRecord(collection)

	// Set basic fields
	record.Set("amount", transaction.Amount)
	record.Set("description", transaction.Description)
	record.Set("date", transaction.Date)
	record.Set("type", string(transaction.Type))
	record.Set("status", string(transaction.Status))
	record.Set("category", transaction.CategoryID)
	record.Set("wallet", transaction.WalletID)

	// Set transfer-specific fields
	if transaction.Type == models.TransactionTypeTransfer {
		record.Set("destination_wallet", transaction.DestWalletID)
		record.Set("exchange_rate", transaction.ExchangeRate)
	}

	// Set tags if present
	if len(transaction.Tags) > 0 {
		record.Set("tags", transaction.Tags)
	}

	// Handle ID (only set for existing records)
	if transaction.ID != "" {
		record.Id = transaction.ID
	}

	return record
}

func (r *TransactionRepository) updateRecordFromTransaction(record *core.Record, transaction *models.Transaction) *core.Record {
	// Update fields
	record.Set("amount", transaction.Amount)
	record.Set("description", transaction.Description)
	record.Set("date", transaction.Date)
	record.Set("type", string(transaction.Type))
	record.Set("status", string(transaction.Status))
	record.Set("category", transaction.CategoryID)
	record.Set("wallet", transaction.WalletID)

	// Update transfer-specific fields
	if transaction.Type == models.TransactionTypeTransfer {
		record.Set("destination_wallet", transaction.DestWalletID)
		record.Set("exchange_rate", transaction.ExchangeRate)
	} else {
		// Clear transfer fields if not a transfer
		record.Set("destination_wallet", "")
		record.Set("exchange_rate", 0)
	}

	// Update tags if present
	if len(transaction.Tags) > 0 {
		record.Set("tags", transaction.Tags)
	}

	return record
}
