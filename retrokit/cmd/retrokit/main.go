package main

import (
	"fmt"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"

	// Register every built-in machine.
	_ "github.com/carlosrabelo/retrokit/retrokit/internal/machine/all"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	first := os.Args[1]
	rest := os.Args[2:]

	switch first {
	case "machines":
		listMachines()
		return
	case "help", "-h", "--help":
		usage()
		return
	}

	if m, err := machine.Lookup(first); err == nil {
		runMachineCommand(m, rest)
		return
	}

	fmt.Fprintf(os.Stderr, "unknown command or machine: %s\n\n", first)
	usage()
	os.Exit(1)
}
