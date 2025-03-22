package main

import (
	"log"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	hooks "github.com/ZanzyTHEbar/firedragon-go/pb_hooks"
)

func main() {
	app := pocketbase.New()

	// Register migrations
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Dir:         "pb_migrations",
		Automigrate: isGoRun,
	})

	// Register hooks
	hooks.RegisterTransactionHooks(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
} 