package repositories

import (
	"context"

	"github.com/ZanzyTHEbar/firedragon-go/domain/models"
)

// CategoryRepository defines the interface for category data access
type CategoryRepository interface {
	// FindByID finds a category by ID
	FindByID(ctx context.Context, id string) (*models.Category, error)

	// FindAll finds all categories with optional filters
	FindAll(ctx context.Context, filter CategoryFilter) ([]*models.Category, error)

	// Create creates a new category
	Create(ctx context.Context, category *models.Category) error

	// Update updates an existing category
	Update(ctx context.Context, category *models.Category) error

	// Delete deletes a category by ID
	Delete(ctx context.Context, id string) error

	// FindByType finds categories by type
	FindByType(ctx context.Context, categoryType models.CategoryType) ([]*models.Category, error)

	// FindSystemCategories finds all system categories
	FindSystemCategories(ctx context.Context) ([]*models.Category, error)
}

// CategoryFilter defines filters for finding categories
type CategoryFilter struct {
	Type       models.CategoryType
	NameLike   string
	IsSystem   *bool
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
} 