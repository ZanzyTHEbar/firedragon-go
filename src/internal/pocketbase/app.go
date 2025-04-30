package pocketbase

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterHooks is currently unused as hooks are registered directly in main.go
// func RegisterHooks(app *pocketbase.PocketBase) error {
//  // Hooks are now registered via pb_hooks.RegisterTransactionHooks in main.go
//  return nil
// }

// RegisterRoutes registers all custom API routes
func RegisterRoutes(app *pocketbase.PocketBase) error {
	// Register custom API routes using OnBeforeServe hook
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Example: Add a custom /api/hello endpoint
		e.Router.GET("/api/hello", func(c echo.Context) error {
			return c.JSON(200, map[string]string{
				"message": "Hello from FireDragon API",
			})
		})

		// TODO: Add more custom API endpoints here

		return nil
	})

	return nil
}

// --- Removed registerTransactionHooks and placeholder functions ---
// --- Logic is now handled in pb_hooks/transaction_hooks.go and domain/usecases ---
