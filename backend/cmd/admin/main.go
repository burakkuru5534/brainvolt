// Command admin runs operator tasks without an API, such as granting staff
// roles (usage: admin grant-role|revoke-role <user> <role>).
// It only wires and starts; everything else lives in internal/app.
package main

import (
	"os"

	"github.com/burakkuru5534/brainvolt/backend/internal/app"
)

func main() {
	os.Exit(app.AdminMain(os.Args[1:], os.Stdout))
}
