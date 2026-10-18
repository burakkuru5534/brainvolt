// Command seed validates and imports the question files under content/
// (usage: seed validate|import|opentdb-fetch, see `seed` without arguments).
// It only wires and starts; everything else lives in internal/app.
package main

import (
	"os"

	"github.com/burakkuru5534/brainvolt/backend/internal/app"
)

func main() {
	os.Exit(app.SeedMain(os.Args[1:], os.Stdout))
}
