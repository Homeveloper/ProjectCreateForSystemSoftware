// Command cfgtool — консольная утилита проверки и преобразования конфигураций.
package main

import (
	"fmt"
	"os"

	"github.com/Homeveloper/ProjectCreateForSystemSoftware/internal/cli"
)

// version подставляется при сборке через -ldflags.
var version = "dev"

func main() {
	if err := cli.Run(os.Args[1:], version); err != nil {
		fmt.Fprintf(os.Stderr, "cfgtool: %v\n", err)
		os.Exit(cli.ExitCodeFor(err))
	}
}
