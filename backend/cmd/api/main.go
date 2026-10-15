// Command api runs the BrainVolt HTTP REST + WebSocket server (the game server).
// It only wires and starts; everything else lives in internal/app.
package main

import (
	"os"

	"github.com/burakkuru5534/brainvolt/backend/internal/app"
)

func main() {
	os.Exit(app.Main(app.NewAPI))
}
