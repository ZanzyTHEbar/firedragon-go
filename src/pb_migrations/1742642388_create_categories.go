package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create categories collection
		categories := core.NewCollection("categories", core.CollectionTypeBase)

		// Add fields
		categories.Fields.Add(
			&core.TextField{
				Name:     "name",
				Required: true,
			},
			&core.TextField{
				Name:     "description",
				Required: false,
			},
			&core.SelectField{
				Name:      "type",
				Required:  true,
				Values:    []string{"income", "expense", "transfer"},
				MaxSelect: 1,
			},
			&core.TextField{
				Name:     "color",
				Required: false,
			},
			&core.SelectField{
				Name:      "is_system",
				Required:  true,
				Values:    []string{"true", "false"},
				MaxSelect: 1,
			},
		)

		if err := app.Save(categories); err != nil {
			return err
		}

		// Get and delete the transactions collection
		transactions, err := app.FindCollectionByNameOrId("transactions")
		if err != nil {
			return err
		}

		if err := app.Delete(transactions); err != nil {
			return err
		}

		// Recreate transactions collection with updated fields
		newTransactions := core.NewCollection("transactions", core.CollectionTypeBase)
		newTransactions.Fields.Add(
			&core.NumberField{
				Name:     "amount",
				Required: true,
				Min:      nil,
			},
			&core.TextField{
				Name:     "description",
				Required: true,
			},
			&core.DateField{
				Name:     "date",
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
				Name:     "exchange_rate",
				Required: false,
				Min:      nil,
			},
			&core.RelationField{
				Name:         "category",
				Required:     true,
				CollectionId: "categories",
				MaxSelect:    1,
			},
			&core.SelectField{
				Name:      "status",
				Required:  true,
				Values:    []string{"pending", "completed", "failed"},
				MaxSelect: 1,
			},
		)

		return app.Save(newTransactions)
	}, func(app core.App) error {
		// Delete the categories collection
		categories, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		if err := app.Delete(categories); err != nil {
			return err
		}

		// Get and delete the transactions collection
		transactions, err := app.FindCollectionByNameOrId("transactions")
		if err != nil {
			return err
		}

		if err := app.Delete(transactions); err != nil {
			return err
		}

		// Recreate transactions collection with original fields
		newTransactions := core.NewCollection("transactions", core.CollectionTypeBase)
		newTransactions.Fields.Add(
			&core.NumberField{
				Name:     "amount",
				Required: true,
				Min:      nil,
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
				Name:     "exchange_rate",
				Required: false,
				Min:      nil,
			},
			&core.SelectField{
				Name:      "status",
				Required:  true,
				Values:    []string{"pending", "completed", "failed"},
				MaxSelect: 1,
			},
		)

		return app.Save(newTransactions)
	})
} 