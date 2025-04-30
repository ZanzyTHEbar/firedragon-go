package models

import (
	"testing"
	"time"
)

func TestNewCategory(t *testing.T) {
	name := "Test Category"
	description := "A category for testing"
	categoryType := CategoryTypeExpense
	color := "#FFFFFF" // Add a placeholder color

	cat := NewCategory(name, description, categoryType, color)

	if cat.ID == "" {
		t.Error("Expected new category to have an ID, but it was empty")
	}
	if cat.Name != name {
		t.Errorf("Expected category name to be '%s', but got '%s'", name, cat.Name)
	}
	if cat.Description != description {
		t.Errorf("Expected category description to be '%s', but got '%s'", description, cat.Description)
	}
	if cat.Type != categoryType {
		t.Errorf("Expected category type to be '%s', but got '%s'", categoryType, cat.Type)
	}
	if cat.IsSystem {
		t.Error("Expected new category IsSystem to be false, but got true")
	}
	if cat.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set, but it was zero")
	}
	if cat.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set, but it was zero")
	}
}

func TestCategory_Validate(t *testing.T) {
	tests := []struct {
		name      string
		category  *Category
		expectErr bool
		errType   error // Expected error type if expectErr is true
	}{
		{
			name: "Valid Category",
			category: &Category{
				ID:        "cat-1",
				Name:      "Groceries",
				Type:      CategoryTypeExpense,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectErr: false,
		},
		{
			name: "Missing Name",
			category: &Category{
				ID:        "cat-2",
				Name:      "", // Missing name
				Type:      CategoryTypeIncome,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectErr: true,
			errType:   ErrMissingCategoryName,
		},
		{
			name: "Invalid Type",
			category: &Category{
				ID:        "cat-3",
				Name:      "Salary",
				Type:      "invalid-type", // Invalid type
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectErr: true,
			errType:   ErrInvalidCategoryType,
		},
		{
			name: "Valid System Category", // System categories bypass some rules if needed (currently same rules)
			category: &Category{
				ID:        "sys-cat-1",
				Name:      "System Income",
				Type:      CategoryTypeIncome,
				IsSystem:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.category.Validate()
			hasErr := err != nil

			if hasErr != tt.expectErr {
				t.Errorf("Validate() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if tt.expectErr && err != tt.errType {
				t.Errorf("Validate() error type = %T, want %T (%v)", err, tt.errType, tt.errType)
			}
		})
	}
}
