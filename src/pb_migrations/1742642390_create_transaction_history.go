package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create transaction history collection
		collection := core.NewCollection("transaction_history", core.CollectionTypeBase)

		// Add fields
		collection.Fields.Add(
			&core.RelationField{
				Name:         "transaction",
				Required:     true,
				CollectionId: "transactions",
				MaxSelect:    1,
			},
			&core.SelectField{
				Name:      "action",
				Required:  true,
				Values:    []string{"created", "updated", "deleted"},
				MaxSelect: 1,
			},
			&core.TextField{
				Name:     "changes",
				Required: true,
			},
			&core.RelationField{
				Name:         "performed_by",
				Required:     false,
				CollectionId: "users",
				MaxSelect:    1,
			},
			&core.DateField{
				Name:     "performed_at",
				Required: true,
			},
			&core.NumberField{
				Name:     "old_balance",
				Required: true,
				Min:      nil,
				Max:      nil,
			},
			&core.NumberField{
				Name:     "new_balance",
				Required: true,
				Min:      nil,
				Max:      nil,
			},
			&core.RelationField{
				Name:         "wallet",
				Required:     true,
				CollectionId: "wallets",
				MaxSelect:    1,
			},
			&core.RelationField{
				Name:         "destination_wallet",
				Required:     false,
				CollectionId: "wallets",
				MaxSelect:    1,
			},
			&core.NumberField{
				Name:     "old_destination_balance",
				Required: false,
				Min:      nil,
				Max:      nil,
			},
			&core.NumberField{
				Name:     "new_destination_balance",
				Required: false,
				Min:      nil,
				Max:      nil,
			},
		)

		// Add indexes
		collection.Indexes = []string{
			"CREATE INDEX idx_transaction_history_transaction ON transaction_history (transaction)",
			"CREATE INDEX idx_transaction_history_action ON transaction_history (action)",
			"CREATE INDEX idx_transaction_history_performed_at ON transaction_history (performed_at)",
			"CREATE INDEX idx_transaction_history_wallet ON transaction_history (wallet)",
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Get and delete the collection
		collection, err := app.FindCollectionByNameOrId("transaction_history")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
} 