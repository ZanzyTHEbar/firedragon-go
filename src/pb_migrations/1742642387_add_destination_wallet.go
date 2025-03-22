package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Get the transactions collection
		collection, err := app.FindCollectionByNameOrId("transactions")
		if err != nil {
			return err
		}

		// Add destination wallet field
		collection.Fields.Add(
			&core.RelationField{
				Name:         "destination_wallet",
				Required:     false, // Only required for transfers
				CollectionId: "wallets",
				MaxSelect:    1,
			},
		)

		// Add exchange rate field for multi-currency transfers
		collection.Fields.Add(
			&core.NumberField{
				Name:     "exchange_rate",
				Required: false,
				Min:      nil,
				Max:      nil,
			},
		)

		return app.Save(collection)
	}, func(app core.App) error {
		// Revert changes by deleting the collection and recreating it
		collection, err := app.FindCollectionByNameOrId("transactions")
		if err != nil {
			return err
		}

		if err := app.Delete(collection); err != nil {
			return err
		}

		// Create a new collection without the fields
		newCollection := core.NewCollection("transactions", core.CollectionTypeBase)
		newCollection.Fields.Add(
			&core.NumberField{
				Name:     "amount",
				Required: true,
			},
			&core.TextField{
				Name:     "description",
				Required: true,
			},
			&core.DateField{
				Name:     "date",
				Required: true,
			},
			&core.TextField{
				Name:     "category",
				Required: true,
			},
			&core.SelectField{
				Name:      "type",
				Required:  true,
				Values:    []string{"income", "expense", "transfer"},
				MaxSelect: 1,
			},
			&core.RelationField{
				Name:         "wallet",
				Required:    true,
				CollectionId: "wallets",
				MaxSelect:   1,
			},
			&core.SelectField{
				Name:      "status",
				Required:  true,
				Values:    []string{"pending", "completed", "failed"},
				MaxSelect: 1,
			},
		)

		return app.Save(newCollection)
	})
} 