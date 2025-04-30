package main

import (
	"log"
	"os"
	"strings"

	pbRepo "github.com/ZanzyTHEbar/firedragon-go/adapters/repositories/pocketbase"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	pbInternal "github.com/ZanzyTHEbar/firedragon-go/internal/pocketbase"
	hooks "github.com/ZanzyTHEbar/firedragon-go/pb_hooks"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	// Initialize app
	app := pocketbase.New()

	// Initialize global logger
	internal.InitGlobalLogger()
	logger := internal.GetLogger()
	logger.Info().Msg("Starting FireDragon server...")

	// Register migrations
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Dir:         "pb_migrations",
		Automigrate: isGoRun,
	})

	// Create repositories
	log.Println("[INFO] Initializing repositories...")
	repoFactory := pbRepo.NewRepositoryFactory(app)
	walletRepo := repoFactory.CreateWalletRepository()
	categoryRepo := repoFactory.CreateCategoryRepository()
	transactionRepo := repoFactory.CreateTransactionRepository()
	log.Println("[INFO] Repositories initialized successfully")

	// Register hooks with repository dependencies
	log.Println("[INFO] Registering transaction hooks...")
	hooks.RegisterTransactionHooks(app, walletRepo, categoryRepo, transactionRepo)

	// Register custom API routes
	log.Println("[INFO] Registering custom API routes...")
	if err := pbInternal.RegisterRoutes(app); err != nil {
		logger.Fatal().Err(err).Msg("Failed to register custom routes")
	}
	log.Println("[INFO] Server initialization complete")

	// Start the application
	logger.Info().Msg("Starting PocketBase server...")
	if err := app.Start(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}
}
