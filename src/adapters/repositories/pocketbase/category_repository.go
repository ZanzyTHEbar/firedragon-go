package pocketbase

import (
	"context"
	"fmt"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/domain/models"
	"github.com/ZanzyTHEbar/firedragon-go/domain/repositories"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// CategoryRepository is a PocketBase implementation of the CategoryRepository interface
type CategoryRepository struct {
	app *pocketbase.PocketBase // Ensure this is the concrete type
}

// NewCategoryRepository creates a new PocketBase category repository
func NewCategoryRepository(app *pocketbase.PocketBase) *CategoryRepository {
	return &CategoryRepository{
		app: app,
	}
}

// FindByID finds a category by ID
func (r *CategoryRepository) FindByID(ctx context.Context, id string) (*models.Category, error) {
	record, err := r.app.FindRecordById("categories", id) // Use r.app directly
	if err != nil {
		return nil, fmt.Errorf("failed to find category: %w", err)
	}

	return r.mapRecordToCategory(record)
}

// FindAll finds all categories with optional filters
func (r *CategoryRepository) FindAll(ctx context.Context, filter repositories.CategoryFilter) ([]*models.Category, error) {
	query := r.app.RecordQuery("categories") // Use r.app directly

	// Apply filters
	if filter.Type != "" {
		query = query.AndWhere(dbx.HashExp{"type": string(filter.Type)})
	}

	if filter.NameLike != "" {
		query = query.AndWhere(dbx.NewExp("name LIKE {:name}", dbx.Params{"name": "%" + filter.NameLike + "%"}))
	}

	if filter.IsSystem != nil {
		query = query.AndWhere(dbx.HashExp{"is_system": *filter.IsSystem})
	}

	// Apply sorting
	if filter.SortBy != "" {
		direction := "ASC"
		if filter.SortOrder == "desc" {
			direction = "DESC"
		}
		query = query.OrderBy(fmt.Sprintf("%s %s", filter.SortBy, direction))
	} else {
		// Default sort by name ascending
		query = query.OrderBy("name ASC")
	}

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(int64(filter.Limit)) // Cast to int64
	}

	if filter.Offset > 0 {
		query = query.Offset(int64(filter.Offset)) // Cast to int64
	}

	// Execute query
	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, fmt.Errorf("failed to find categories: %w", err)
	}

	// Convert records to domain models
	categories := make([]*models.Category, 0, len(records))
	for _, record := range records {
		category, err := r.mapRecordToCategory(record)
		if err != nil {
			return nil, fmt.Errorf("failed to map record to category: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// Create creates a new category
func (r *CategoryRepository) Create(ctx context.Context, category *models.Category) error {
	record := r.mapCategoryToRecord(category)

	if err := r.app.Save(record); err != nil { // Use r.app directly
		return fmt.Errorf("failed to create category: %w", err)
	}

	// Update the category ID from the saved record
	category.ID = record.Id

	return nil
}

// Update updates an existing category
func (r *CategoryRepository) Update(ctx context.Context, category *models.Category) error {
	// Check if category exists
	record, err := r.app.FindRecordById("categories", category.ID) // Use r.app directly
	if err != nil {
		return fmt.Errorf("failed to find category: %w", err)
	}

	// Prevent updates to system categories except by admin
	if record.GetBool("is_system") && !isAdminContext(ctx) {
		return models.ErrSystemCategoryCannotBeDeleted
	}

	// Update fields
	record = r.updateRecordFromCategory(record, category)

	if err := r.app.Save(record); err != nil { // Use r.app directly
		return fmt.Errorf("failed to update category: %w", err)
	}

	return nil
}

// Delete deletes a category by ID
func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	record, err := r.app.FindRecordById("categories", id) // Use r.app directly
	if err != nil {
		return fmt.Errorf("failed to find category: %w", err)
	}

	// Prevent deletion of system categories
	if record.GetBool("is_system") {
		return models.ErrSystemCategoryCannotBeDeleted
	}

	// Check for any transactions using this category
	var txCount int64 // Use int64 for count
	// Select count(*) and use Row() to scan the result
	countQuery := r.app.RecordQuery("transactions").Select("count(*)").AndWhere(dbx.HashExp{"category": id})
	if err := countQuery.Row(&txCount); err != nil { // Use Row() to get the count
		return fmt.Errorf("failed to check for transactions using category: %w", err)
	}

	if txCount > 0 {
		return fmt.Errorf("category cannot be deleted because it has %d transactions", txCount)
	}

	// If no transactions reference this category, delete it
	if err := r.app.Delete(record); err != nil { // Use r.app directly
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}

// FindByType finds categories by type
func (r *CategoryRepository) FindByType(ctx context.Context, categoryType models.CategoryType) ([]*models.Category, error) {
	query := r.app.RecordQuery("categories"). // Use r.app directly
		AndWhere(dbx.HashExp{"type": string(categoryType)}).
		OrderBy("name ASC")

	// Execute query
	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, fmt.Errorf("failed to find categories by type: %w", err)
	}

	// Convert records to domain models
	categories := make([]*models.Category, 0, len(records))
	for _, record := range records {
		category, err := r.mapRecordToCategory(record)
		if err != nil {
			return nil, fmt.Errorf("failed to map record to category: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// FindSystemCategories finds all system categories
func (r *CategoryRepository) FindSystemCategories(ctx context.Context) ([]*models.Category, error) {
	query := r.app.RecordQuery("categories"). // Use r.app directly
		AndWhere(dbx.HashExp{"is_system": true}).
		OrderBy("name ASC")

	// Execute query
	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, fmt.Errorf("failed to find system categories: %w", err)
	}

	// Convert records to domain models
	categories := make([]*models.Category, 0, len(records))
	for _, record := range records {
		category, err := r.mapRecordToCategory(record)
		if err != nil {
			return nil, fmt.Errorf("failed to map record to category: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// Helper methods for mapping between domain models and PocketBase records

func (r *CategoryRepository) mapRecordToCategory(record *core.Record) (*models.Category, error) {
	category := &models.Category{
		ID:          record.Id,
		Name:        record.GetString("name"),
		Description: record.GetString("description"),
		Type:        models.CategoryType(record.GetString("type")),
		Color:       record.GetString("color"),
		IsSystem:    record.GetBool("is_system"),
		CreatedAt:   record.GetDateTime("created").Time(),
		UpdatedAt:   record.GetDateTime("updated").Time(),
	}

	return category, nil
}

func (r *CategoryRepository) mapCategoryToRecord(category *models.Category) *core.Record {
	collection, _ := r.app.FindCollectionByNameOrId("categories") // Use r.app directly
	record := core.NewRecord(collection)

	// Set basic fields
	record.Set("name", category.Name)
	record.Set("description", category.Description)
	record.Set("type", string(category.Type))
	record.Set("color", category.Color)
	record.Set("is_system", category.IsSystem)

	// Set ID if specified
	if category.ID != "" {
		record.Id = category.ID
	}

	// Update timestamps if not set
	if category.CreatedAt.IsZero() {
		category.CreatedAt = time.Now()
	}
	category.UpdatedAt = time.Now()

	return record
}

func (r *CategoryRepository) updateRecordFromCategory(record *core.Record, category *models.Category) *core.Record {
	// Update fields
	record.Set("name", category.Name)
	record.Set("description", category.Description)
	record.Set("type", string(category.Type))
	record.Set("color", category.Color)
	// Don't update is_system flag from regular updates

	return record
}

// Check if context represents an admin operation
// This is a placeholder - replace with actual logic for admin auth
func isAdminContext(ctx context.Context) bool {
	// TODO: Implement proper admin authorization check
	return false
}
