package pb_hooks

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// TransactionValidator handles transaction validation logic
type TransactionValidator struct {
	app *pocketbase.PocketBase
}

// ValidateTransaction performs validation checks on a transaction
func (v *TransactionValidator) ValidateTransaction(record *core.Record) error {
	// Validate amount
	amount := record.GetFloat("amount")
	if amount <= 0 {
		return fmt.Errorf("invalid amount: must be greater than 0")
	}

	// Validate date
	date := record.GetDateTime("date")
	if date.Time().After(time.Now()) {
		return fmt.Errorf("invalid date: cannot be in the future")
	}

	// Get source wallet
	wallet, err := getWallet(v.app, record.GetString("wallet"))
	if err != nil {
		return fmt.Errorf("failed to get source wallet: %v", err)
	}

	// Get and validate category
	category, err := getCategory(v.app, record.GetString("category"))
	if err != nil {
		return fmt.Errorf("failed to get category: %v", err)
	}

	// Validate category type matches transaction type
	transType := record.GetString("type")
	categoryType := category.GetString("type")
	if transType != categoryType {
		return fmt.Errorf("category type '%s' does not match transaction type '%s'", categoryType, transType)
	}

	// Validate based on transaction type
	switch transType {
	case "expense":
		// Validate wallet has sufficient balance for expense
		balance := wallet.GetFloat("balance")
		if balance < amount {
			return fmt.Errorf("insufficient balance in source wallet")
		}

	case "transfer":
		// Validate destination wallet exists
		destWalletID := record.GetString("destination_wallet")
		if destWalletID == "" {
			return fmt.Errorf("destination wallet is required for transfers")
		}

		destWallet, err := getWallet(v.app, destWalletID)
		if err != nil {
			return fmt.Errorf("failed to get destination wallet: %v", err)
		}

		// Validate source wallet has sufficient balance
		balance := wallet.GetFloat("balance")
		if balance < amount {
			return fmt.Errorf("insufficient balance in source wallet")
		}

		// If wallets have different currencies, validate exchange rate
		if wallet.GetString("currency") != destWallet.GetString("currency") {
			exchangeRate := record.GetFloat("exchange_rate")
			if exchangeRate <= 0 {
				return fmt.Errorf("exchange rate is required for cross-currency transfers")
			}
		}

		// Prevent self-transfers
		if wallet.Id == destWallet.Id {
			return fmt.Errorf("cannot transfer to the same wallet")
		}
	}

	return nil
}

// TransactionProcessor handles transaction processing logic
type TransactionProcessor struct {
	app *pocketbase.PocketBase
}

// ProcessTransaction handles the business logic for transaction processing
func (p *TransactionProcessor) ProcessTransaction(record *core.Record, oldRecord *core.Record) error {
	// Run the wallet update in a transaction to ensure consistency
	return p.app.RunInTransaction(func(txApp core.App) error {
		// Get source wallet
		wallet, err := getWallet(txApp, record.GetString("wallet"))
		if err != nil {
			return fmt.Errorf("failed to get source wallet: %v", err)
		}

		// Update wallet balance
		amount := record.GetFloat("amount")
		currentBalance := wallet.GetFloat("balance")
		destOldBalance := 0.0
		var destWallet *core.Record

		// If this is an update, first reverse the old transaction
		if oldRecord != nil {
			oldAmount := oldRecord.GetFloat("amount")
			oldType := oldRecord.GetString("type")
			
			// Reverse the old transaction effect
			switch oldType {
			case "income":
				currentBalance -= oldAmount
			case "expense":
				currentBalance += oldAmount
			case "transfer":
				// Reverse source wallet
				currentBalance += oldAmount

				// Reverse destination wallet
				oldDestWallet, err := getWallet(txApp, oldRecord.GetString("destination_wallet"))
				if err != nil {
					return fmt.Errorf("failed to get old destination wallet: %v", err)
				}

				destOldBalance = oldDestWallet.GetFloat("balance")
				oldExchangeRate := oldRecord.GetFloat("exchange_rate")
				if oldExchangeRate > 0 {
					oldDestWallet.Set("balance", destOldBalance-oldAmount*oldExchangeRate)
				} else {
					oldDestWallet.Set("balance", destOldBalance-oldAmount)
				}

				if err := txApp.Save(oldDestWallet); err != nil {
					return fmt.Errorf("failed to update old destination wallet balance: %v", err)
				}
			}
		}

		// Apply the new transaction
		switch record.GetString("type") {
		case "income":
			wallet.Set("balance", currentBalance+amount)
		case "expense":
			wallet.Set("balance", currentBalance-amount)
		case "transfer":
			// Update source wallet
			wallet.Set("balance", currentBalance-amount)

			// Update destination wallet
			destWallet, err = getWallet(txApp, record.GetString("destination_wallet"))
			if err != nil {
				return fmt.Errorf("failed to get destination wallet: %v", err)
			}

			destBalance := destWallet.GetFloat("balance")
			destOldBalance = destBalance
			exchangeRate := record.GetFloat("exchange_rate")
			if exchangeRate > 0 {
				destWallet.Set("balance", destBalance+amount*exchangeRate)
			} else {
				destWallet.Set("balance", destBalance+amount)
			}

			if err := txApp.Save(destWallet); err != nil {
				return fmt.Errorf("failed to update destination wallet balance: %v", err)
			}
		}

		// Save source wallet changes
		if err := txApp.Save(wallet); err != nil {
			return fmt.Errorf("failed to update source wallet balance: %v", err)
		}

		return nil
	})
}

// ReverseTransaction reverses the effects of a transaction
func (p *TransactionProcessor) ReverseTransaction(record *core.Record) error {
	// Run the wallet update in a transaction to ensure consistency
	return p.app.RunInTransaction(func(txApp core.App) error {
		// Get source wallet
		wallet, err := getWallet(txApp, record.GetString("wallet"))
		if err != nil {
			return fmt.Errorf("failed to get source wallet: %v", err)
		}

		// Get current balance and amount
		amount := record.GetFloat("amount")
		currentBalance := wallet.GetFloat("balance")
		oldBalance := currentBalance
		destOldBalance := 0.0
		var destWallet *core.Record

		// Reverse the transaction effect based on type
		switch record.GetString("type") {
		case "income":
			wallet.Set("balance", currentBalance-amount)
		case "expense":
			wallet.Set("balance", currentBalance+amount)
		case "transfer":
			// Reverse source wallet
			wallet.Set("balance", currentBalance+amount)

			// Reverse destination wallet
			destWallet, err = getWallet(txApp, record.GetString("destination_wallet"))
			if err != nil {
				return fmt.Errorf("failed to get destination wallet: %v", err)
			}

			destBalance := destWallet.GetFloat("balance")
			destOldBalance = destBalance
			exchangeRate := record.GetFloat("exchange_rate")
			if exchangeRate > 0 {
				destWallet.Set("balance", destBalance-amount*exchangeRate)
			} else {
				destWallet.Set("balance", destBalance-amount)
			}

			if err := txApp.Save(destWallet); err != nil {
				return fmt.Errorf("failed to update destination wallet balance: %v", err)
			}
		}

		// Save source wallet changes
		if err := txApp.Save(wallet); err != nil {
			return fmt.Errorf("failed to update source wallet balance: %v", err)
		}

		// Record the deletion in history
		if err := p.recordHistory(txApp, record, "deleted", nil, wallet, oldBalance, destWallet, destOldBalance); err != nil {
			return fmt.Errorf("failed to record history: %v", err)
		}

		return nil
	})
}

// recordHistory creates a history record for a transaction change
func (p *TransactionProcessor) recordHistory(app core.App, record *core.Record, action string, oldRecord *core.Record, wallet *core.Record, oldBalance float64, destWallet *core.Record, destOldBalance float64) error {
	// Create history record
	historyCollection, err := app.FindCollectionByNameOrId("transaction_history")
	if err != nil {
		return fmt.Errorf("failed to find history collection: %v", err)
	}

	history := core.NewRecord(historyCollection)
	history.Set("transaction", record.Id)
	history.Set("action", action)
	history.Set("performed_at", time.Now())
	history.Set("wallet", wallet.Id)
	history.Set("old_balance", oldBalance)
	history.Set("new_balance", wallet.GetFloat("balance"))

	// Set destination wallet info if applicable
	if destWallet != nil {
		history.Set("destination_wallet", destWallet.Id)
		history.Set("old_destination_balance", destOldBalance)
		history.Set("new_destination_balance", destWallet.GetFloat("balance"))
	}

	// Record changes
	changes := map[string]interface{}{
		"new": record.PublicExport(),
	}
	if oldRecord != nil {
		changes["old"] = oldRecord.PublicExport()
	}
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %v", err)
	}
	history.Set("changes", string(changesJSON))

	// Save the history record
	if err := app.Save(history); err != nil {
		return fmt.Errorf("failed to save history record: %v", err)
	}

	return nil
}

// RegisterTransactionHooks registers all transaction-related hooks
func RegisterTransactionHooks(app *pocketbase.PocketBase) {
	validator := &TransactionValidator{app: app}
	processor := &TransactionProcessor{app: app}

	// Before create
	app.OnModelCreate().BindFunc(func(e *core.ModelEvent) error {
		record, ok := e.Model.(*core.Record)
		if !ok || record.Collection().Name != "transactions" {
			return e.Next()
		}

		if err := validator.ValidateTransaction(record); err != nil {
			return err
		}

		// Check for duplicate transactions
		exists, err := checkDuplicateTransaction(app, record)
		if err != nil {
			return fmt.Errorf("failed to check for duplicates: %v", err)
		}
		if exists {
			return fmt.Errorf("duplicate transaction detected")
		}

		return e.Next()
	})

	// Before update
	app.OnModelUpdate().BindFunc(func(e *core.ModelEvent) error {
		record, ok := e.Model.(*core.Record)
		if !ok || record.Collection().Name != "transactions" {
			return e.Next()
		}

		if err := validator.ValidateTransaction(record); err != nil {
			return err
		}

		return e.Next()
	})

	// Before delete
	app.OnModelDelete().BindFunc(func(e *core.ModelEvent) error {
		record, ok := e.Model.(*core.Record)
		if !ok || record.Collection().Name != "transactions" {
			return e.Next()
		}

		// Reverse the transaction effects before deletion
		if err := processor.ReverseTransaction(record); err != nil {
			return fmt.Errorf("failed to reverse transaction: %v", err)
		}

		return e.Next()
	})

	// After create
	app.OnModelAfterCreateSuccess().BindFunc(func(e *core.ModelEvent) error {
		record, ok := e.Model.(*core.Record)
		if !ok || record.Collection().Name != "transactions" {
			return e.Next()
		}

		if err := processor.ProcessTransaction(record, nil); err != nil {
			log.Printf("Error processing transaction: %v", err)
			record.Set("status", "failed")
			if err := app.Save(record); err != nil {
				log.Printf("Failed to update transaction status: %v", err)
			}
			return e.Next()
		}

		// Record the creation in history
		wallet, err := getWallet(app, record.GetString("wallet"))
		if err != nil {
			log.Printf("Failed to get wallet for history: %v", err)
			return e.Next()
		}

		var destWallet *core.Record
		if record.GetString("type") == "transfer" {
			destWallet, err = getWallet(app, record.GetString("destination_wallet"))
			if err != nil {
				log.Printf("Failed to get destination wallet for history: %v", err)
				return e.Next()
			}
		}

		if err := processor.recordHistory(app, record, "created", nil, wallet, 0, destWallet, 0); err != nil {
			log.Printf("Failed to record history: %v", err)
			return e.Next()
		}

		record.Set("status", "completed")
		if err := app.Save(record); err != nil {
			log.Printf("Failed to update transaction status: %v", err)
		}

		return e.Next()
	})

	// After update
	app.OnModelAfterUpdateSuccess().BindFunc(func(e *core.ModelEvent) error {
		record, ok := e.Model.(*core.Record)
		if !ok || record.Collection().Name != "transactions" {
			return e.Next()
		}

		// Get the old record to reverse its effects
		oldRecord, err := app.FindRecordById("transactions", record.Id)
		if err != nil {
			log.Printf("Failed to find old transaction record: %v", err)
			return e.Next()
		}

		if err := processor.ProcessTransaction(record, oldRecord); err != nil {
			log.Printf("Error processing updated transaction: %v", err)
			record.Set("status", "failed")
			if err := app.Save(record); err != nil {
				log.Printf("Failed to update transaction status: %v", err)
			}
			return e.Next()
		}

		// Record the update in history
		wallet, err := getWallet(app, record.GetString("wallet"))
		if err != nil {
			log.Printf("Failed to get wallet for history: %v", err)
			return e.Next()
		}

		var destWallet *core.Record
		if record.GetString("type") == "transfer" {
			destWallet, err = getWallet(app, record.GetString("destination_wallet"))
			if err != nil {
				log.Printf("Failed to get destination wallet for history: %v", err)
				return e.Next()
			}
		}

		if err := processor.recordHistory(app, record, "updated", oldRecord, wallet, wallet.GetFloat("balance"), destWallet, destWallet.GetFloat("balance")); err != nil {
			log.Printf("Failed to record history: %v", err)
			return e.Next()
		}

		record.Set("status", "completed")
		if err := app.Save(record); err != nil {
			log.Printf("Failed to update transaction status: %v", err)
		}

		return e.Next()
	})
}

// checkDuplicateTransaction checks if a similar transaction exists within a time window
func checkDuplicateTransaction(app *pocketbase.PocketBase, record *core.Record) (bool, error) {
	amount := record.GetFloat("amount")
	date := record.GetDateTime("date")
	walletId := record.GetString("wallet")
	transType := record.GetString("type")
	categoryId := record.GetString("category")

	// Check for transactions within 24 hours with same amount, wallet, type, and category
	startDate := date.Time().Add(-24 * time.Hour)
	endDate := date.Time().Add(24 * time.Hour)

	// Query for duplicate transactions
	records := []*core.Record{}
	err := app.RecordQuery("transactions").
		AndWhere(dbx.HashExp{
			"amount":   amount,
			"wallet":   walletId,
			"type":     transType,
			"category": categoryId,
		}).
		AndWhere(dbx.NewExp("date >= {:startDate}", dbx.Params{"startDate": startDate})).
		AndWhere(dbx.NewExp("date <= {:endDate}", dbx.Params{"endDate": endDate})).
		All(&records)

	if err != nil {
		return false, fmt.Errorf("failed to check for duplicates: %v", err)
	}

	// For transfers, also check destination wallet
	if transType == "transfer" {
		destWalletId := record.GetString("destination_wallet")
		for _, r := range records {
			if r.GetString("destination_wallet") == destWalletId {
				return true, nil
			}
		}
		return false, nil
	}

	return len(records) > 0, nil
}

func getWallet(app core.App, walletId string) (*core.Record, error) {
	record, err := app.FindRecordById("wallets", walletId)
	if err != nil {
		return nil, fmt.Errorf("invalid wallet: %v", err)
	}
	return record, nil
}

func getCategory(app core.App, categoryId string) (*core.Record, error) {
	record, err := app.FindRecordById("categories", categoryId)
	if err != nil {
		return nil, fmt.Errorf("invalid category: %v", err)
	}
	return record, nil
}

