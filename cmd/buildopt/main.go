package main

import (
	"os"

	"github.com/tonyredondo/buildopt/internal/launcher"
)

func main() {
	os.Exit(launcher.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
