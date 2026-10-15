// Command worker runs the BrainVolt background and scheduled jobs.
// It only wires and starts; everything else lives in internal/app.
package main

import (
	"os"

	"github.com/burakkuru5534/brainvolt/backend/internal/app"
)

func main() {
	os.Exit(app.Main(app.NewWorker))
}
