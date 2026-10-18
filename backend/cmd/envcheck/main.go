// Command envcheck compares a developer's .env with .env.example and lists
// missing variables and ${VAR} references that would expand to nothing
// (usage: envcheck <.env.example> <.env>). `make check-env` runs it before
// up, run, migrate and test.
package main

import (
	"fmt"
	"os"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: envcheck <.env.example> <.env>")
		os.Exit(2)
	}
	example, err := config.ReadEnvFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "envcheck:", err)
		os.Exit(1)
	}
	actual, err := config.ReadEnvFile(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "envcheck:", err)
		os.Exit(1)
	}
	problems := config.CheckEnvFile(example, actual, os.LookupEnv)
	if len(problems) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "%s needs fixing (compare it with %s):\n", os.Args[2], os.Args[1])
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "  -", p)
	}
	os.Exit(1)
}
