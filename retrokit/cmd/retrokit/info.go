package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/info"
	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
)

func runInfo(m machine.Machine, args []string) {
	fs := flag.NewFlagSet(m.Name()+" info", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: retrokit %s info <file>\n", m.Name())
		os.Exit(1)
	}

	result, err := info.Inspect(fs.Arg(0), m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(result.String())
}
