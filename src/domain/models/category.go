package models

import (
	"time"

	"github.com/google/uuid"
)

// CategoryType defines the type of category
type CategoryType string

const (
	// CategoryTypeIncome represents an income category
	CategoryTypeIncome CategoryType = "income"
	
	// CategoryTypeExpense represents an expense category
	CategoryTypeExpense CategoryType = "expense"
	
	// CategoryTypeTransfer represents a transfer category
	CategoryTypeTransfer CategoryType = "transfer"
)

// Category represents a transaction category
type Category struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        CategoryType `json:"type"`
	Color       string       `json:"color"`
	IsSystem    bool         `json:"isSystem"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// NewCategory creates a new category with defaults
func NewCategory(name, description string, categoryType CategoryType, color string) *Category {
	return &Category{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Type:        categoryType,
		Color:       color,
		IsSystem:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// NewSystemCategory creates a new system category
func NewSystemCategory(name, description string, categoryType CategoryType, color string) *Category {
	category := NewCategory(name, description, categoryType, color)
	category.IsSystem = true
	return category
}

// Validate checks if the category is valid
func (c *Category) Validate() error {
	// Name is required
	if c.Name == "" {
		return ErrMissingCategoryName
	}
	
	// Type must be valid
	if c.Type != CategoryTypeIncome && 
	   c.Type != CategoryTypeExpense && 
	   c.Type != CategoryTypeTransfer {
		return ErrInvalidCategoryType
	}
	
	return nil
}

// MatchesTransactionType checks if the category type matches the transaction type
func (c *Category) MatchesTransactionType(txType TransactionType) bool {
	switch txType {
	case TransactionTypeIncome:
		return c.Type == CategoryTypeIncome
	case TransactionTypeExpense:
		return c.Type == CategoryTypeExpense
	case TransactionTypeTransfer:
		return c.Type == CategoryTypeTransfer
	default:
		return false
	}
} 