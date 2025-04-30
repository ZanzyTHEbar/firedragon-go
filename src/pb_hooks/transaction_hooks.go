package pb_hooks

import (
	"log" // Keep standard log for now

	// Import necessary domain packages
	"github.com/ZanzyTHEbar/firedragon-go/domain/models"
	"github.com/ZanzyTHEbar/firedragon-go/domain/repositories"
	"github.com/ZanzyTHEbar/firedragon-go/domain/usecases"

	// Import PocketBase packages
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterTransactionHooks registers simplified transaction hooks.
// It accepts repository dependencies (currently unused in this simplified version).
func RegisterTransactionHooks(
	app *pocketbase.PocketBase,
	walletRepo repositories.WalletRepository,
	categoryRepo repositories.CategoryRepository,
	transactionRepo repositories.TransactionRepository,
) {
	log.Println("[INFO] Registering simplified PocketBase transaction hooks...")

	// Create transaction service
	transactionService := usecases.NewTransactionService(walletRepo, categoryRepo, transactionRepo)

	// Use Model Hook: OnModelCreate with BindFunc and filter by collection name
	app.OnModelCreate("transactions").BindFunc(func(e *core.ModelEvent) error {
		log.Printf("[Hook OnModelBeforeCreate] Triggered for transactions")
		
		// Validate transaction record
		// TODO: Add validation before creating transaction
		
		return nil
	})

	// Use Model Hook: OnModelUpdate with BindFunc and filter by collection name
	app.OnModelUpdate("transactions").BindFunc(func(e *core.ModelEvent) error {
		record, ok := e.Model.(*core.Record)
		if !ok {
			return nil
		}
		log.Printf("[Hook OnModelBeforeUpdate] Triggered for transactions ID: %s", record.Id)
		
		// Validate transaction update
		// TODO: Add validation before updating transaction
		
		return nil
	})

	// Use Model Hook: OnModelDelete with BindFunc and filter by collection name
	app.OnModelDelete("transactions").BindFunc(func(e *core.ModelEvent) error {
		record, ok := e.Model.(*core.Record)
		if !ok {
			return nil
		}
		log.Printf("[Hook OnModelBeforeDelete] Triggered for transactions ID: %s", record.Id)
		
		// TODO: Implement transaction deletion logic
		
		return nil
	})

	// Use Model Hook: OnModelAfterCreateSuccess with BindFunc and filter by collection name
	app.OnModelAfterCreateSuccess("transactions").BindFunc(func(e *core.ModelEvent) error {
		record, ok := e.Model.(*core.Record)
		if !ok {
			return nil
		}
		log.Printf("[Hook OnModelAfterCreateSuccess] Triggered for transactions ID: %s", record.Id)
		
		// Convert PocketBase record to domain model input
		input := mapRecordToTransactionInput(record)
		
		// Process transaction using service
		_, err := transactionService.CreateTransaction(input)
		if err != nil {
			log.Printf("[ERROR] Failed to process transaction: %v", err)
			return err
		}
		
		log.Printf("[INFO] Successfully processed transaction ID: %s", record.Id)
		return nil
	})

	// Use Model Hook: OnModelAfterUpdateSuccess with BindFunc and filter by collection name
	app.OnModelAfterUpdateSuccess("transactions").BindFunc(func(e *core.ModelEvent) error {
		record, ok := e.Model.(*core.Record)
		if !ok {
			return nil
		}
		log.Printf("[Hook OnModelAfterUpdateSuccess] Triggered for transactions ID: %s", record.Id)
		
		// TODO: Implement transaction update logic using service
		
		return nil
	})

	log.Println("[INFO] PocketBase transaction hooks registration attempt complete.")
}

// mapRecordToTransactionInput converts a PocketBase record to a CreateTransactionInput
func mapRecordToTransactionInput(record *core.Record) usecases.CreateTransactionInput {
	// Extract required data from the record
	amount := record.GetFloat("amount")
	description := record.GetString("description")
	date := record.GetDateTime("date").Time()
	txType := models.TransactionType(record.GetString("type"))
	categoryID := record.GetString("category")
	walletID := record.GetString("wallet")
	
	// Initialize optional fields
	var destWalletID string
	var exchangeRate float64
	
	// Handle optional fields based on transaction type
	if txType == models.TransactionTypeTransfer {
		// For transfer transactions, get destination wallet if available
		destWalletID = record.GetString("destination_wallet")
		
		// Get exchange rate or use default value of 1.0
		exchangeRate = record.GetFloat("exchange_rate")
		if exchangeRate == 0 {
			// Default exchange rate for transfers is 1.0 if not specified or invalid
			exchangeRate = 1.0
		}
	}
	
	// Handle tags if available
	var tags []string
	tagValues := record.Get("tags")
	if tagSlice, ok := tagValues.([]any); ok {
		for _, tag := range tagSlice {
			if tagStr, ok := tag.(string); ok && tagStr != "" {
				tags = append(tags, tagStr)
			}
		}
	}
	
	// Create input struct with all available fields
	return usecases.CreateTransactionInput{
		Amount:       amount,
		Description:  description,
		Date:         date,
		Type:         txType,
		CategoryID:   categoryID,
		WalletID:     walletID,
		DestWalletID: destWalletID,
		ExchangeRate: exchangeRate,
		Tags:         tags,
	}
}
