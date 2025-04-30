package pocketbase

import (
	"encoding/json"
	"net/http"

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
	// Register custom API routes using OnServe hook with BindFunc
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Example: Add a custom /api/hello endpoint
		e.Router.GET("/api/hello", func(c *core.RequestEvent) error {
			type response struct {
				Message string `json:"message"`
			}

			data, err := json.Marshal(&response{Message: "Hello from FireDragon API"})
			if err != nil {
				return err // Return error for proper handling
			}

			c.Response.Header().Set("Content-Type", "application/json")
			c.Response.WriteHeader(http.StatusOK)
			_, err = c.Response.Write(data)
			return err // Return potential write error
		})

		// TODO: Add more custom API endpoints here

		return e.Next() // Call e.Next() to proceed with the hook chain
	})

	return nil
}

// --- Removed registerTransactionHooks and placeholder functions ---
// --- Logic is now handled in pb_hooks/transaction_hooks.go and domain/usecases ---
