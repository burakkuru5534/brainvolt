// Command migrate applies the database migrations in backend/migrations
// (usage: migrate up|down|status|version).
// It only wires and starts; everything else lives in internal/app.
package main

import (
	"os"

	"github.com/burakkuru5534/brainvolt/backend/internal/app"
)

func main() {
	os.Exit(app.MigrateMain(os.Args[1:], os.Stdout))
}
