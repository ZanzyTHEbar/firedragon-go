package pb_migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Get the categories collection
		collection, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		// Create system categories
		systemCategories := []struct {
			name        string
			description string
			type_       string
			color       string
			is_system   string
		}{
			// Income categories
			{
				name:        "Salary",
				description: "Regular employment income",
				type_:       "income",
				color:       "#4CAF50", // Green
				is_system:   "true",
			},
			{
				name:        "Investment",
				description: "Income from investments",
				type_:       "income",
				color:       "#2196F3", // Blue
				is_system:   "true",
			},
			{
				name:        "Other Income",
				description: "Miscellaneous income",
				type_:       "income",
				color:       "#9C27B0", // Purple
				is_system:   "true",
			},

			// Expense categories
			{
				name:        "Housing",
				description: "Rent, mortgage, and housing expenses",
				type_:       "expense",
				color:       "#F44336", // Red
				is_system:   "true",
			},
			{
				name:        "Transportation",
				description: "Car, public transport, and travel expenses",
				type_:       "expense",
				color:       "#FF9800", // Orange
				is_system:   "true",
			},
			{
				name:        "Food",
				description: "Groceries and dining out",
				type_:       "expense",
				color:       "#795548", // Brown
				is_system:   "true",
			},
			{
				name:        "Utilities",
				description: "Electricity, water, internet, etc.",
				type_:       "expense",
				color:       "#607D8B", // Blue Grey
				is_system:   "true",
			},
			{
				name:        "Healthcare",
				description: "Medical and health-related expenses",
				type_:       "expense",
				color:       "#E91E63", // Pink
				is_system:   "true",
			},
			{
				name:        "Entertainment",
				description: "Recreation and entertainment expenses",
				type_:       "expense",
				color:       "#673AB7", // Deep Purple
				is_system:   "true",
			},
			{
				name:        "Other Expenses",
				description: "Miscellaneous expenses",
				type_:       "expense",
				color:       "#757575", // Grey
				is_system:   "true",
			},

			// Transfer categories
			{
				name:        "Internal Transfer",
				description: "Transfer between own accounts",
				type_:       "transfer",
				color:       "#009688", // Teal
				is_system:   "true",
			},
			{
				name:        "External Transfer",
				description: "Transfer to external accounts",
				type_:       "transfer",
				color:       "#00BCD4", // Cyan
				is_system:   "true",
			},
		}

		// Create each system category
		for _, cat := range systemCategories {
			record := core.NewRecord(collection)
			record.Set("name", cat.name)
			record.Set("description", cat.description)
			record.Set("type", cat.type_)
			record.Set("color", cat.color)
			record.Set("is_system", cat.is_system)

			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Get the categories collection
		collection, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		// Find and delete all system categories
		records := []*core.Record{}
		err = app.RecordQuery(collection.Name).
			AndWhere(dbx.HashExp{"is_system": "true"}).
			All(&records)

		if err != nil {
			return err
		}

		for _, record := range records {
			if err := app.Delete(record); err != nil {
				return err
			}
		}

		return nil
	})
} 