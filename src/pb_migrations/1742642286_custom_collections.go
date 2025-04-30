package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// Create transactions collection
		transactions := core.NewCollection("transactions", core.CollectionTypeBase)

		// Add fields
		transactions.Fields.Add(
			&core.NumberField{
				Name:     "amount",
				Required: true,
				Min:      types.Pointer(0.0),
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
				Name:     "type",
				Required: true,
				Values:   []string{"income", "expense", "transfer"},
				MaxSelect: 1,
			},
			&core.RelationField{
				Name:     "wallet",
				Required: true,
				CollectionId: "wallets",
				MaxSelect: 1,
			},
			&core.SelectField{
				Name:     "status",
				Required: true,
				Values:   []string{"pending", "completed", "failed"},
				MaxSelect: 1,
			},
		)

		if err := app.Save(transactions); err != nil {
			return err
		}

		// Create wallets collection
		wallets := core.NewCollection("wallets", core.CollectionTypeBase)

		// Add fields
		wallets.Fields.Add(
			&core.TextField{
				Name:     "name",
				Required: true,
			},
			&core.NumberField{
				Name:     "balance",
				Required: true,
				Min:      types.Pointer(0.0),
			},
			&core.TextField{
				Name:     "currency",
				Required: true,
			},
			&core.SelectField{
				Name:     "type",
				Required: true,
				Values:   []string{"bank", "crypto", "cash"},
				MaxSelect: 1,
			},
		)

		if err := app.Save(wallets); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Delete collections (in reverse order to handle relations)
		collections := []string{"transactions", "wallets"}
		for _, name := range collections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			if err := app.Delete(collection); err != nil {
				return err
			}
		}

		return nil
	})
} 